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
		defer conn.Close()

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

	pools, metadataBytes, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 3) // 3 assets -> 3 pairs

	for _, p := range pools {
		assert.Less(t, p.Tokens[0].Address, p.Tokens[1].Address, "pair tokens must be sorted")
	}

	// Second call, same assets: everything already seen, no new pools.
	pools2, _, err := u.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	assert.Empty(t, pools2)
}
