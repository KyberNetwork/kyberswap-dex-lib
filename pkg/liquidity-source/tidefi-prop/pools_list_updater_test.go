package tidefiprop

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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
	// Non-quoting pairs are re-probed once their backoff elapsed.
	asked = nil
	u.run += backoffBase
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

// fakeChain stubs the RPC seams and counts calls. A pair quotes iff quoting
// returns true for its pool address.
type fakeChain struct {
	decCalls   map[string]int // token -> times decimals was read
	decMulti   int            // decimals multicalls
	quoteMulti []int          // quote calls per multicall
	badDec     map[string]bool
	quoting    func(addr string) bool
}

func newFakeUpdater(f *fakeChain, tokens []string) *PoolsListUpdater {
	f.decCalls = map[string]int{}
	u := NewPoolsListUpdater(&Config{DexID: DexType, Tokens: tokens}, nil)
	u.readDecimals = func(_ context.Context, toks []string) ([]uint8, error) {
		f.decMulti++
		out := make([]uint8, len(toks))
		for i, t := range toks {
			f.decCalls[t]++
			if !f.badDec[t] {
				out[i] = 18
			}
		}
		return out, nil
	}
	u.callQuotes = func(_ context.Context, calls []quoteCall) ([]bool, error) {
		f.quoteMulti = append(f.quoteMulti, len(calls))
		out := make([]bool, len(calls))
		for i, c := range calls {
			t0, t1 := c.in, c.out
			if t0 > t1 {
				t0, t1 = t1, t0
			}
			out[i] = f.quoting(poolAddress(t0, t1))
		}
		return out, nil
	}
	return u
}

// Decimals never change: each token is read once across runs, and a failed
// read is retried (not cached) while its pairs stay unlisted.
func TestPoolsListUpdater_DecimalsCached(t *testing.T) {
	t.Parallel()

	f := &fakeChain{badDec: map[string]bool{"0xc": true}, quoting: func(string) bool { return true }}
	u := newFakeUpdater(f, []string{"0xa", "0xb", "0xc"})

	pools, md, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1, "only a/b has decimals for both tokens")
	assert.Equal(t, poolAddress("0xa", "0xb"), pools[0].Address)

	u.run += backoffBase
	f.badDec = nil // c now answers
	pools, _, err = u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	assert.Len(t, pools, 2)
	assert.Equal(t, map[string]int{"0xa": 1, "0xb": 1, "0xc": 2}, f.decCalls)
}

// A dead pair is not re-probed until its backoff elapses, then retried with a
// doubled delay (capped); once it quotes it is listed and the backoff is gone.
func TestPoolsListUpdater_Backoff(t *testing.T) {
	t.Parallel()

	live := false
	f := &fakeChain{quoting: func(string) bool { return live }}
	u := newFakeUpdater(f, []string{"0xa", "0xb"})
	probes := func() int { n := len(f.quoteMulti); f.quoteMulti = nil; return n }
	runOnce := func(md []byte) ([]entity.Pool, []byte) {
		p, m, err := u.GetNewPools(context.Background(), md)
		require.NoError(t, err)
		return p, m
	}

	_, md := runOnce(nil) // run 1: probed, next = 1+base
	assert.Equal(t, 1, probes())
	for r := 2; r < 1+backoffBase; r++ {
		runOnce(md)
		require.Zero(t, probes(), "run %d must skip the dead pair", r)
	}
	runOnce(md) // run 1+base: due again, delay doubles
	assert.Equal(t, 1, probes())
	assert.Equal(t, 2*backoffBase, u.backoff[poolAddress("0xa", "0xb")].delay)
	for r := 0; r < 2*backoffBase-1; r++ {
		runOnce(md)
		require.Zero(t, probes())
	}
	live = true
	pools, md := runOnce(md)
	assert.Equal(t, 1, probes())
	require.Len(t, pools, 1)
	assert.Empty(t, u.backoff, "quoting pair leaves the backoff set")

	// Listed pair is never probed again.
	for r := 0; r < 3*backoffBase; r++ {
		runOnce(md)
	}
	assert.Zero(t, probes())
}

// Backoff delay resets to base when a pair that was backed off quotes, then
// stops: it is Seen, so no new backoff entry appears. A pair that keeps
// failing caps at backoffMax.
func TestPoolsListUpdater_BackoffCap(t *testing.T) {
	t.Parallel()

	f := &fakeChain{quoting: func(string) bool { return false }}
	u := newFakeUpdater(f, []string{"0xa", "0xb"})
	var md []byte
	var err error
	for range 12 {
		u.run = max(u.run, u.backoff[poolAddress("0xa", "0xb")].next-1)
		_, md, err = u.GetNewPools(context.Background(), md)
		require.NoError(t, err)
	}
	assert.Equal(t, backoffMax, u.backoff[poolAddress("0xa", "0xb")].delay)
}

// parityQuoting makes tokens of equal index parity quote against each other
// (about half of all pairs), so no live token looks dead during a sweep.
func parityQuoting(toks []string) func(string) bool {
	q := map[string]bool{}
	for i := range toks {
		for j := i + 2; j < len(toks); j += 2 {
			q[poolAddress(toks[i], toks[j])] = true
		}
	}
	return func(a string) bool { return q[a] }
}

func synthTokens(n int) []string {
	toks := make([]string, n)
	for i := range toks {
		toks[i] = fmt.Sprintf("0x%04x", i)
	}
	return toks
}

// 100 tokens = 4950 pairs. Each run probes at most probeBudget pairs in
// multicalls of at most probeChunk pairs; successive runs cover every pair
// exactly once, and dead pairs are only re-probed after their backoff.
func TestPoolsListUpdater_BudgetAndCoverage(t *testing.T) {
	t.Parallel()

	toks := synthTokens(100)
	quoting := parityQuoting(toks)
	f := &fakeChain{quoting: quoting}
	u := newFakeUpdater(f, toks)
	var asked [][2]string
	inner := u.probe
	u.probe = func(ctx context.Context, pairs [][2]string) ([]bool, error) {
		require.LessOrEqual(t, len(pairs), probeChunk)
		asked = append(asked, pairs...)
		return inner(ctx, pairs)
	}

	total := len(toks) * (len(toks) - 1) / 2
	runs := (total + probeBudget - 1) / probeBudget
	var md []byte
	listed := 0
	cover := map[[2]string]int{}
	for r := 1; r <= runs; r++ {
		asked, f.quoteMulti = nil, nil
		pools, m, err := u.GetNewPools(context.Background(), md)
		require.NoError(t, err)
		md = m
		listed += len(pools)
		require.LessOrEqual(t, len(asked), probeBudget)
		require.LessOrEqual(t, len(f.quoteMulti), probeBudget/probeChunk)
		for _, n := range f.quoteMulti {
			require.LessOrEqual(t, n, 2*probeChunk)
		}
		for _, p := range asked {
			cover[p]++
		}
	}
	assert.Len(t, cover, total, "every pair reached within %d runs", runs)
	for p, n := range cover {
		require.Equal(t, 1, n, "pair %v probed %d times in the first sweep", p, n)
	}
	assert.Equal(t, 2*(50*49/2), listed, "every same-parity pair is listed")
	assert.Equal(t, len(toks), len(f.decCalls))
	for tok, n := range f.decCalls {
		require.Equal(t, 1, n, tok)
	}

	// Dead pairs wait out the backoff: nothing is due until run 1+backoffBase.
	for u.run < backoffBase {
		asked = nil
		_, _, err := u.GetNewPools(context.Background(), md)
		require.NoError(t, err)
		require.Empty(t, asked, "run %d", u.run)
	}
	asked = nil
	_, _, err := u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	assert.NotEmpty(t, asked, "dead pairs are retried once their backoff elapsed")
	assert.LessOrEqual(t, len(asked), probeBudget)
}

// New tokens (never-probed pairs) are probed before due backoff retries.
func TestPoolsListUpdater_NewPairsFirst(t *testing.T) {
	t.Parallel()

	toks := synthTokens(30) // 435 pairs, about half quote
	f := &fakeChain{quoting: parityQuoting(toks)}
	u := newFakeUpdater(f, toks)
	var md []byte
	for r := 0; r < 3; r++ { // 600 > 435: every pair probed once
		_, m, err := u.GetNewPools(context.Background(), md)
		require.NoError(t, err)
		md = m
	}
	u.run = 100 // every dead pair is due
	var asked [][2]string
	inner := u.probe
	u.probe = func(ctx context.Context, p [][2]string) ([]bool, error) {
		asked = append(asked, p...)
		return inner(ctx, p)
	}
	toks = append(toks, "0xffff")
	u.cfg.Tokens = toks
	_, _, err := u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	require.Len(t, asked, probeBudget)
	for _, p := range asked[:30] {
		assert.Equal(t, "0xffff", p[1], "new token pairs come first")
	}
}

// A token with no listed pair that failed against deadAfter partners is
// skipped, except on sample runs, so it stops costing probes.
func TestPoolsListUpdater_DeadToken(t *testing.T) {
	t.Parallel()

	toks := synthTokens(40)
	dead := toks[len(toks)-1] // last, so its pairs spread over the first sweep
	f := &fakeChain{quoting: func(a string) bool { return !strings.Contains(a, dead) }}
	u := newFakeUpdater(f, toks)
	var asked [][2]string
	inner := u.probe
	u.probe = func(ctx context.Context, p [][2]string) ([]bool, error) {
		asked = append(asked, p...)
		return inner(ctx, p)
	}
	var md []byte
	deadProbes := func() (n int) {
		for _, p := range asked {
			if p[0] == dead || p[1] == dead {
				n++
			}
		}
		return
	}
	for u.run < deadSampleEvery-1 { // 780 pairs = 4 runs; backoff retries start at run 31
		_, m, err := u.GetNewPools(context.Background(), md)
		require.NoError(t, err)
		md = m
	}
	// The dead token's 39 pairs were probed at most about deadAfter times
	// (plus the rest of its first chunk) instead of 39 per backoff cycle.
	assert.Less(t, deadProbes(), 20)
	assert.GreaterOrEqual(t, u.failed[dead], deadAfter)

	asked = nil
	_, _, err := u.GetNewPools(context.Background(), md) // run == deadSampleEvery
	require.NoError(t, err)
	assert.Positive(t, deadProbes(), "sample run probes the dead token again")
}
