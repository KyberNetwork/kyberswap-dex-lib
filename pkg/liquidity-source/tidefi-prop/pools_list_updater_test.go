package tidefiprop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTakerAPIServer serves one tidefi_markets push per connection, requiring
// the given bearer token, then blocks on the next read until the client
// disconnects (mirroring the real API, which keeps the socket open).
func newTakerAPIServer(t *testing.T, token string, assetAddrs []string) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		assets := make([]map[string]string, 0, len(assetAddrs))
		for _, a := range assetAddrs {
			assets = append(assets, map[string]string{"token_address": a})
		}
		msg, err := json.Marshal(map[string]any{
			"method": "tidefi_markets",
			"params": map[string]any{"assets": assets},
		})
		require.NoError(t, err)
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, msg))

		_, _, _ = conn.ReadMessage() // block until client disconnects
	}))
}

// quotingProbe stubs the on-chain probe: a pair quotes iff its pool address is
// in quoting. It records every pair it is asked about.
func quotingProbe(asked *[][2]string, quoting ...string) func(context.Context, [][2]string) ([]bool, error) {
	return func(_ context.Context, pairs [][2]string) ([]bool, error) {
		*asked = append(*asked, pairs...)
		out := make([]bool, len(pairs))
		for i, p := range pairs {
			for _, q := range quoting {
				out[i] = out[i] || poolAddress(p[0], p[1]) == q
			}
		}
		return out, nil
	}
}

func TestPoolsListUpdater_GetNewPools(t *testing.T) {
	t.Parallel()

	const token = "test-token"
	assets := []string{
		"0xB000000000000000000000000000000000000B",
		"0xA000000000000000000000000000000000000A",
		"0xC000000000000000000000000000000000000C",
	}
	srv := newTakerAPIServer(t, token, assets)
	defer srv.Close()

	cfg := &Config{
		DexID:       DexType,
		Address:     "0xSwapper",
		TakerAPIURL: "ws" + strings.TrimPrefix(srv.URL, "http") + "/",
		AuthToken:   token,
	}
	u := NewPoolsListUpdater(cfg, nil)
	var asked [][2]string
	u.probe = quotingProbe(&asked,
		poolAddress(strings.ToLower(assets[1]), strings.ToLower(assets[0])),
		poolAddress(strings.ToLower(assets[0]), strings.ToLower(assets[2])),
		poolAddress(strings.ToLower(assets[1]), strings.ToLower(assets[2])))

	pools, metadataBytes, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 3) // 3 assets -> 3 pairs
	require.Len(t, asked, 3)

	for _, p := range pools {
		assert.Less(t, p.Tokens[0].Address, p.Tokens[1].Address, "pair tokens must be sorted")
	}

	// Second call, same assets: everything already seen, no new pools.
	asked = nil
	pools2, _, err := u.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	assert.Empty(t, pools2)
	assert.Empty(t, asked, "seen pairs must not be re-probed")
}

// Only pairs that quote are listed; a pair that does not quote stays out of
// Seen so the next call re-probes it (and lists it once it quotes), while
// listed pairs are never re-probed.
func TestPoolsListUpdater_GetNewPools_ProbeFilter(t *testing.T) {
	t.Parallel()

	cfg := &Config{DexID: DexType, Address: "0xSwapper", Tokens: []string{"0xa", "0xb", "0xc"}}
	u := NewPoolsListUpdater(cfg, nil)
	var asked [][2]string
	u.probe = quotingProbe(&asked, poolAddress("0xa", "0xb"))

	pools, md, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	assert.Equal(t, poolAddress("0xa", "0xb"), pools[0].Address)
	assert.Len(t, asked, 3, "static Tokens must be probed too")

	// 0xa/0xc starts quoting; 0xa/0xb is already listed, 0xb/0xc still empty.
	asked = nil
	u.probe = quotingProbe(&asked, poolAddress("0xa", "0xc"))
	pools, _, err = u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	assert.Equal(t, poolAddress("0xa", "0xc"), pools[0].Address)
	assert.ElementsMatch(t, [][2]string{{"0xa", "0xc"}, {"0xb", "0xc"}}, asked)
}

func TestPoolsListUpdater_GetNewPools_ProbeError(t *testing.T) {
	t.Parallel()

	u := NewPoolsListUpdater(&Config{DexID: DexType, Tokens: []string{"0xa", "0xb"}}, nil)
	u.probe = func(context.Context, [][2]string) ([]bool, error) { return nil, assert.AnError }
	_, md, err := u.GetNewPools(context.Background(), []byte(`{"seen":{}}`))
	require.ErrorIs(t, err, assert.AnError)
	assert.JSONEq(t, `{"seen":{}}`, string(md))
}

// A static token list must replace Taker API discovery entirely: on chains
// where the API is unreachable, any dial attempt would fail the whole update.
func TestPoolsListUpdater_GetNewPools_StaticTokens(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		DexID:       DexType,
		Address:     "0xSwapper",
		TakerAPIURL: "ws://127.0.0.1:1/", // unreachable; must not be dialed
		Tokens: []string{
			"0xB000000000000000000000000000000000000B",
			"0xA000000000000000000000000000000000000A",
			"0xC000000000000000000000000000000000000C",
		},
	}
	u := NewPoolsListUpdater(cfg, nil)
	var asked [][2]string
	u.probe = quotingProbe(&asked,
		poolAddress("0xa000000000000000000000000000000000000a", "0xb000000000000000000000000000000000000b"),
		poolAddress("0xa000000000000000000000000000000000000a", "0xc000000000000000000000000000000000000c"),
		poolAddress("0xb000000000000000000000000000000000000b", "0xc000000000000000000000000000000000000c"))

	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 3)

	for _, p := range pools {
		assert.Less(t, p.Tokens[0].Address, p.Tokens[1].Address, "pair tokens must be sorted")
		assert.Equal(t, strings.ToLower(p.Tokens[0].Address), p.Tokens[0].Address, "tokens must be lowercased")
	}
}
