package oneinchaqua

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// The Portal tests use a fake Portal over TLS (the config requires https) and the fake node of the lister
// tests. They share the process-wide metric hook, so they are not parallel.

const testPortalKey = "sk-portal-TOP-SECRET-7f3a91"

// fakePortal serves /aqua/v1.0/strategies/opened for one chain from a list of items.
type fakePortal struct {
	*httptest.Server
	mu    sync.Mutex
	items []map[string]any
	// fault, when set, may answer a request itself (call is the 1-based request number); it returns true when it did.
	fault func(call int, w http.ResponseWriter, r *http.Request) bool
	// onPage runs before each page is served, under no lock the test holds.
	onPage  func(call int)
	calls   int
	queries []string
	auths   []string
	paths   []string
}

func newFakePortal(t *testing.T, items ...map[string]any) *fakePortal {
	fp := &fakePortal{items: items}
	fp.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp.mu.Lock()
		fp.calls++
		call := fp.calls
		fp.queries = append(fp.queries, r.URL.RawQuery)
		fp.auths = append(fp.auths, r.Header.Get("Authorization"))
		fp.paths = append(fp.paths, r.URL.Path)
		fault, onPage, all := fp.fault, fp.onPage, append([]map[string]any{}, fp.items...)
		fp.mu.Unlock()
		if onPage != nil {
			onPage(call)
		}
		if fault != nil && fault(call, w, r) {
			return
		}
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		offset, _ := strconv.Atoi(q.Get("cursor"))
		if limit <= 0 || r.Header.Get("Authorization") != "Bearer "+testPortalKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		end := min(offset+limit, len(all))
		page := map[string]any{"items": all[min(offset, len(all)):end], "total": len(all)}
		if end < len(all) {
			page["nextCursor"] = strconv.Itoa(end)
		}
		_ = json.NewEncoder(w).Encode(page)
	}))
	t.Cleanup(fp.Close)
	return fp
}

func (fp *fakePortal) callCount() int {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return fp.calls
}

// portalItemOf is a Portal item for an order: the Portal sorts tokens ascending and sends cached balances,
// which the lister must ignore (these are absurd on purpose).
func portalItemOf(chainId uint64, o *Order, app common.Address, tokens ...string) map[string]any {
	toks := make([]map[string]any, len(tokens))
	for i, tk := range tokens {
		toks[i] = map[string]any{"address": tk, "balance": map[string]any{"strategy": "999999999999", "wallet": "1"}, "allowance": "0"}
	}
	return map[string]any{"chainId": chainId, "maker": addrHex(o.Maker), "app": addrHex(app), "strategyHash": o.Hash().Hex(),
		"strategyBytes": hexutil.Encode(o.Encode()), "strategyBytesSize": len(o.Encode()), "openedAt": 1794135243, "tokens": toks}
}

type logCapture struct {
	mu    sync.Mutex
	lines []string
}

func (c *logCapture) log(level logLevel, msg string, fields logger.Fields) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, fmt.Sprintf("%d %s %v", level, msg, fields))
}

func (c *logCapture) joined() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.lines, "\n")
}

// portalConfig is a lister config with the Portal at fp.
func portalConfig(fp *fakePortal, mutate ...func(*Config)) Config {
	retries := 2
	cfg := Config{DexId: DexType, ChainId: 1, StartBlock: 1, LogWindow: 50, MaxWindowsPerRun: 1, ProbedTokens: probedAll(tokenA, tokenB, tokenC),
		PortalURL: fp.URL, PortalApiKey: testPortalKey, PortalPageLimit: 2, PortalMaxPages: 10,
		PortalTimeout: durationjson.Duration{Duration: 2 * time.Second}, PortalMaxRetries: &retries,
		PortalRetryBackoff: durationjson.Duration{Duration: time.Millisecond}}
	for _, m := range mutate {
		m(&cfg)
	}
	return cfg
}

// newPortalLister builds a lister whose Portal client trusts the fake's certificate.
func newPortalLister(t *testing.T, n *fakeNode, fp *fakePortal, cfg Config) (*PoolsListUpdater, *logCapture) {
	t.Helper()
	conf := 12
	if cfg.Confirmations == nil {
		c := uint64(conf)
		cfg.Confirmations = &c
	}
	base := testConfig()
	cfg.AquaAddress, cfg.RouterAddress = base.AquaAddress, base.RouterAddress
	u, err := NewPoolsListUpdater(&cfg, n.client())
	require.NoError(t, err)
	capture := &logCapture{}
	u.log = capture.log
	if u.portal != nil {
		u.portal.http.SetTransport(fp.Client().Transport)
	}
	return u, capture
}

// metaJSON is stored lister metadata: the cursor and the head of the last full sweep.
func metaJSON(t *testing.T, last, portalHead uint64) []byte {
	b, err := json.Marshal(Metadata{LastScannedBlock: last, LastPortalHead: portalHead})
	require.NoError(t, err)
	return b
}

func metaOf(t *testing.T, meta []byte) Metadata {
	var m Metadata
	require.NoError(t, json.Unmarshal(meta, &m))
	return m
}

func cursorOf(t *testing.T, meta []byte) uint64 { return metaOf(t, meta).LastScannedBlock }

func testOrders(n int) []*Order {
	var out []*Order
	for i := range n {
		out = append(out, orderOf(testMaker, prog(iFlat(uint32(i+1)), iSwap())))
	}
	return out
}

func idsOf(pools []entity.Pool) map[string]entity.Pool {
	m := map[string]entity.Pool{}
	for _, p := range pools {
		m[p.Address] = p
	}
	return m
}

// A full sweep over several pages returns every strategy, reads no logs, moves the cursor to the head of the
// previous full sweep (never back, never past it) and stores the confirmed head read before this sweep.
func TestPortalSweepPaginationAndCursor(t *testing.T) {
	recorder.reset()
	orders := testOrders(5)
	var items []map[string]any
	for _, o := range orders {
		items = append(items, portalItemOf(1, o, testRouter, tokenA, tokenB))
	}
	fp := newFakePortal(t, items...)
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))

	for name, tc := range map[string]struct {
		stored []byte
		want   Metadata
	}{
		"first sweep stores only the head":   {nil, Metadata{0, 988}},
		"old metadata is a first sweep":      {metaJSON(t, 500, 0), Metadata{500, 988}},
		"cursor moves to the previous head":  {metaJSON(t, 500, 700), Metadata{700, 988}},
		"cursor already past the previous":   {metaJSON(t, 800, 700), Metadata{800, 988}},
		"cursor ahead of the head":           {metaJSON(t, 2000, 900), Metadata{2000, 988}},
		"previous head ahead of the new one": {metaJSON(t, 500, 2000), Metadata{2000, 988}},
	} {
		before := fp.callCount()
		pools, meta, err := u.GetNewPools(context.Background(), tc.stored)
		require.NoError(t, err, name)
		require.Len(t, pools, 5, name)
		require.Equal(t, tc.want, metaOf(t, meta), name)
		require.Equal(t, 3, fp.callCount()-before, "%s: 5 items at 2 per page are 3 requests", name)
		require.Empty(t, n.requestLog(), "%s: no log is read after a full sweep", name)
	}

	// The requests: path, chain, limit, the cursor of each page, the key as a bearer token on every one.
	fp.mu.Lock()
	defer fp.mu.Unlock()
	require.Equal(t, portalPath, fp.paths[0])
	require.Equal(t, "chainIds=1&limit=2", fp.queries[0])
	require.Equal(t, "chainIds=1&cursor=2&limit=2", fp.queries[1])
	require.Equal(t, "chainIds=1&cursor=4&limit=2", fp.queries[2])
	for _, a := range fp.auths {
		require.Equal(t, "Bearer "+testPortalKey, a)
	}
}

// The head is read before the sweep: a head that moves during the sweep does not move the cursor.
func TestPortalSweepReadsHeadFirst(t *testing.T) {
	o := testOrders(3)
	fp := newFakePortal(t, portalItemOf(1, o[0], testRouter, tokenA, tokenB), portalItemOf(1, o[1], testRouter, tokenA, tokenB), portalItemOf(1, o[2], testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	fp.onPage = func(call int) {
		n.mu.Lock()
		n.head = 1000 + uint64(call)*100
		n.mu.Unlock()
	}
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	_, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, Metadata{LastPortalHead: 988}, metaOf(t, meta))
}

type faultCase struct {
	name   string
	reason string
	// setup changes the fake portal or the config.
	fault func(call int, w http.ResponseWriter, r *http.Request) bool
	cfg   func(*Config)
}

func statusFault(code int) func(int, http.ResponseWriter, *http.Request) bool {
	return func(_ int, w http.ResponseWriter, _ *http.Request) bool {
		http.Error(w, "nope", code)
		return true
	}
}

func rawBody(body string) func(int, http.ResponseWriter, *http.Request) bool {
	return func(_ int, w http.ResponseWriter, _ *http.Request) bool {
		_, _ = w.Write([]byte(body))
		return true
	}
}

// Every way a sweep fails runs the log path from the stored cursor: the cursor is not advanced from the
// head (the log scan moves it only by what it scanned), the log pool is listed and the reason is counted.
func TestPortalFallbackReasons(t *testing.T) {
	cases := []faultCase{
		{name: "401", reason: portalAuth, fault: statusFault(401)},
		{name: "403", reason: portalAuth, fault: statusFault(403)},
		{name: "429 exhausted", reason: portalRateLimit, fault: statusFault(429)},
		{name: "500", reason: portalServer, fault: statusFault(500)},
		{name: "503", reason: portalServer, fault: statusFault(503)},
		{name: "404", reason: portalStatus, fault: statusFault(404)},
		{name: "malformed json", reason: portalMalformed, fault: rawBody("{not json")},
		{name: "html", reason: portalMalformed, fault: rawBody("<html></html>")},
		{name: "empty object", reason: portalMalformed, fault: rawBody("{}")},
		{name: "null", reason: portalMalformed, fault: rawBody("null")},
		{name: "items null", reason: portalMalformed, fault: rawBody(`{"items":null}`)},
		{name: "error object", reason: portalMalformed, fault: rawBody(`{"message":"quota"}`)},
		{name: "other schema", reason: portalMalformed, fault: rawBody(`{"data":[]}`)},
		{name: "wrong chain", reason: portalWrongChain, fault: func(_ int, w http.ResponseWriter, _ *http.Request) bool {
			o := testOrders(1)[0]
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{portalItemOf(56, o, testRouter, tokenA, tokenB)}})
			return true
		}},
		{name: "cursor does not advance", reason: portalMalformed, fault: func(_ int, w http.ResponseWriter, _ *http.Request) bool {
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB)}, "nextCursor": "x"})
			return true
		}},
		{name: "timeout", reason: portalTimeoutFail, fault: func(_ int, w http.ResponseWriter, r *http.Request) bool {
			select {
			case <-time.After(2 * time.Second):
			case <-r.Context().Done():
			}
			return true
		}, cfg: func(c *Config) { c.PortalTimeout = durationjson.Duration{Duration: 50 * time.Millisecond} }},
		{name: "page cap", reason: portalPageCap, cfg: func(c *Config) { c.PortalMaxPages = 2 }},
		{name: "bad item", reason: portalBadItem, fault: func(_ int, w http.ResponseWriter, _ *http.Request) bool {
			it := portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB)
			it["maker"] = "0xnothex"
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{it}})
			return true
		}},
		{name: "bytes omitted", reason: portalBadItem, fault: func(_ int, w http.ResponseWriter, _ *http.Request) bool {
			it := portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB)
			it["strategyBytes"], it["strategyBytesOmittedReason"] = nil, "too_large"
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{it}})
			return true
		}},
		{name: "redirect", reason: portalNetwork, fault: func(_ int, w http.ResponseWriter, r *http.Request) bool {
			http.Redirect(w, r, "https://example.invalid/steal", http.StatusFound)
			return true
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder.reset()
			var items []map[string]any
			for _, o := range testOrders(5) {
				items = append(items, portalItemOf(1, o, testRouter, tokenA, tokenB))
			}
			fp := newFakePortal(t, items...)
			fp.fault = tc.fault
			n := newFakeNode(t, 1000)
			logOrder := orderOf(common.HexToAddress("0x9999999999999999999999999999999999999999"), prog(iFlat(77), iSwap()))
			ship(t, n, 10, txHash(1), 0, logOrder, testRouter, tokenA, tokenB)
			cfg := portalConfig(fp)
			if tc.cfg != nil {
				tc.cfg(&cfg)
			}
			u, capture := newPortalLister(t, n, fp, cfg)

			pools, meta, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
			require.NoError(t, err)
			require.Equal(t, float64(1), recorder.get(metricPortalFallback+"|"+tc.reason), "the fallback is counted with its reason")
			require.Contains(t, capture.joined(), "fell back to logs")
			require.Contains(t, capture.joined(), tc.reason)

			// The log path ran from the stored cursor, one window: 6..55, and the cursor is where it ended,
			// not the confirmed head 988.
			require.Equal(t, [][2]uint64{{6, 55}}, n.requestLog())
			require.Equal(t, uint64(55), cursorOf(t, meta))
			require.Contains(t, idsOf(pools), poolId(testRouter, logOrder.Maker, logOrder.Hash()), "the log scan lists its strategy")
			require.NotContains(t, capture.joined(), testPortalKey)
			require.NotContains(t, fmt.Sprint(recorder.counts), testPortalKey)
		})
	}
}

const metricPortalFallback = "portal_fallback"

// No key: the Portal is not called, each run counts a no_key fallback and logs are the discovery.
func TestPortalNoKeyFallsBack(t *testing.T) {
	recorder.reset()
	fp := newFakePortal(t, portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	ship(t, n, 10, txHash(1), 0, testOrders(1)[0], testRouter, tokenA, tokenB)
	for _, key := range []string{"", "   "} {
		u, _ := newPortalLister(t, n, fp, portalConfig(fp, func(c *Config) { c.PortalApiKey = key }))
		pools, meta, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
		require.NoError(t, err)
		require.Len(t, pools, 1)
		require.Equal(t, uint64(55), cursorOf(t, meta))
	}
	require.Zero(t, fp.callCount(), "no request without a key")
	require.Equal(t, float64(2), recorder.get(metricPortalFallback+"|"+portalNoKey))

	// No URL: logs only and no fallback is counted: it is the configured mode.
	recorder.reset()
	u := newLister(t, n, Config{StartBlock: 1, LogWindow: 50, MaxWindowsPerRun: 1})
	_, _, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
	require.NoError(t, err)
	require.Zero(t, recorder.get(metricPortalFallback+"|"+portalNoKey))
}

// 429 is retried with a wait and a later success is a full sweep; every request carries the key.
func TestPortalRetries429(t *testing.T) {
	recorder.reset()
	var items []map[string]any
	for _, o := range testOrders(3) {
		items = append(items, portalItemOf(1, o, testRouter, tokenA, tokenB))
	}
	fp := newFakePortal(t, items...)
	fp.fault = func(call int, w http.ResponseWriter, _ *http.Request) bool {
		if call <= 2 { // the first two requests are throttled, the retry budget is 2
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return true
		}
		return false
	}
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	pools, meta, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
	require.NoError(t, err)
	require.Len(t, pools, 3)
	require.Equal(t, Metadata{5, 988}, metaOf(t, meta))
	require.Empty(t, n.requestLog())
	require.Equal(t, 2+2, fp.callCount(), "two throttled requests, then two pages")
	require.Zero(t, recorder.get(metricPortalFallback+"|"+portalRateLimit))

	// With no retries allowed the first 429 is the fallback.
	recorder.reset()
	fp2 := newFakePortal(t, items...)
	fp2.fault = statusFault(429)
	zero := 0
	u, _ = newPortalLister(t, n, fp2, portalConfig(fp2, func(c *Config) { c.PortalMaxRetries = &zero }))
	_, _, err = u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
	require.NoError(t, err)
	require.Equal(t, 1, fp2.callCount())
	require.Equal(t, float64(1), recorder.get(metricPortalFallback+"|"+portalRateLimit))
}

// A connection that cannot be made is a network fallback.
func TestPortalNetworkError(t *testing.T) {
	recorder.reset()
	fp := newFakePortal(t, portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	u, capture := newPortalLister(t, n, fp, portalConfig(fp))
	fp.Close()
	_, meta, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
	require.NoError(t, err)
	require.Equal(t, float64(1), recorder.get(metricPortalFallback+"|"+portalNetwork))
	require.Equal(t, uint64(55), cursorOf(t, meta))
	require.NotContains(t, capture.joined(), testPortalKey)
}

// A sweep that fails part-way returns what its first pages admitted, then the log scan runs: pools listed by
// both paths appear once.
func TestPortalPartialSweepThenLogs(t *testing.T) {
	recorder.reset()
	orders := testOrders(5)
	var items []map[string]any
	for _, o := range orders {
		items = append(items, portalItemOf(1, o, testRouter, tokenA, tokenB))
	}
	fp := newFakePortal(t, items...)
	fp.fault = func(call int, w http.ResponseWriter, _ *http.Request) bool {
		if call == 3 { // pages 1 and 2 (4 items) are served, page 3 fails
			http.Error(w, "boom", http.StatusInternalServerError)
			return true
		}
		return false
	}
	n := newFakeNode(t, 1000)
	ship(t, n, 10, txHash(1), 0, orders[0], testRouter, tokenA, tokenB) // also in the Portal's first page
	extra := orderOf(common.HexToAddress("0x9999999999999999999999999999999999999999"), prog(iFlat(77), iSwap()))
	ship(t, n, 11, txHash(2), 0, extra, testRouter, tokenA, tokenB)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	pools, meta, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
	require.NoError(t, err)
	require.Len(t, pools, 5, "4 from the Portal, 1 more from the logs, the shared one once")
	byId := idsOf(pools)
	require.Len(t, byId, 5)
	require.Contains(t, byId, poolId(testRouter, extra.Maker, extra.Hash()))
	require.NotContains(t, byId, poolId(testRouter, orders[4].Maker, orders[4].Hash()), "the failed page's strategy is not listed")
	require.Equal(t, uint64(55), cursorOf(t, meta), "the cursor is the log scan's, not the head")
	require.Equal(t, float64(1), recorder.get(metricPortalFallback+"|"+portalServer))
}

// A cancelled context is an error, not a fallback.
func TestPortalContextCancelled(t *testing.T) {
	recorder.reset()
	fp := newFakePortal(t, portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	fp.fault = func(_ int, _ http.ResponseWriter, r *http.Request) bool {
		cancel()
		select { // hold the reply until the client has given up
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
		return true
	}
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	stored := metaJSON(t, 5, 0)
	_, meta, err := u.GetNewPools(ctx, stored)
	require.Error(t, err)
	require.Equal(t, stored, meta)
	require.Zero(t, recorder.get(metricPortalFallback+"|"+portalNetwork))
}

// The Portal items go through the same admission as the log path.
func TestPortalAdmission(t *testing.T) {
	recorder.reset()
	other := common.HexToAddress("0x8888888888888888888888888888888888888888")
	foreign := common.HexToAddress("0xf3da3145b208ebfa94fad073ba9c03d6e8746ffe")
	good := orderOf(testMaker, prog(iFlat(1), iSwap()))
	launchOrder := orderOf(testMaker, prog(iFlat(2), iSwap()))
	unsorted := orderOf(testMaker, prog(iFlat(3), iSwap()))
	threeTok := orderOf(testMaker, prog(iFlat(4), iSwap()))
	unprobed := orderOf(testMaker, prog(iFlat(5), iSwap()))
	unsupported := orderOf(testMaker, prog(ins(0x23), iSwap()))
	guarded := orderOf(testMaker, prog(ins(opTxOriginGuard, 1), iSwap()))
	makerMismatch := orderOf(other, prog(iFlat(6), iSwap()))
	foreignApp := orderOf(testMaker, prog(iFlat(7), iSwap()))
	tooFewTok := orderOf(testMaker, prog(iFlat(8), iSwap()))
	sameTok := orderOf(testMaker, prog(iFlat(9), iSwap()))
	hashMismatch := portalItemOf(1, orderOf(testMaker, prog(iFlat(10), iSwap())), testRouter, tokenA, tokenB)
	hashMismatch["strategyHash"] = good.Hash().Hex()
	notAnOrder := portalItemOf(1, orderOf(testMaker, prog(iFlat(11), iSwap())), testRouter, tokenA, tokenB)
	notAnOrder["strategyBytes"] = "0x1234"
	notAnOrder["strategyHash"] = common.Hash{1}.Hex()
	makerItem := portalItemOf(1, makerMismatch, testRouter, tokenA, tokenB)
	makerItem["maker"] = addrHex(testMaker) // the item says another maker than the order inside the bytes
	docked := orderOf(testMaker, prog(iFlat(12), iSwap()))

	items := []map[string]any{
		portalItemOf(1, good, testRouter, tokenA, tokenB),
		portalItemOf(1, launchOrder, testLaunch, tokenA, tokenB),
		portalItemOf(1, unsorted, testRouter, tokenB, tokenA),
		portalItemOf(1, threeTok, testRouter, tokenA, tokenB, tokenC),
		portalItemOf(1, unprobed, testRouter, tokenA, "0x4444444444444444444444444444444444444444"),
		portalItemOf(1, unsupported, testRouter, tokenA, tokenB),
		portalItemOf(1, guarded, testRouter, tokenA, tokenB),
		makerItem,
		portalItemOf(1, foreignApp, foreign, tokenA, tokenB),
		portalItemOf(1, tooFewTok, testRouter, tokenA),
		portalItemOf(1, sameTok, testRouter, tokenA, tokenA),
		hashMismatch,
		notAnOrder,
		portalItemOf(1, docked, testRouter, tokenA, tokenB),
	}
	fp := newFakePortal(t, items...)
	n := newFakeNode(t, 1000)

	// Without the launch router configured, the launch-app strategy is an unknown app.
	u, _ := newPortalLister(t, n, fp, portalConfig(fp, func(c *Config) { c.PortalPageLimit = 100 }))
	pools, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, Metadata{LastPortalHead: 988}, metaOf(t, meta), "discarded items do not fail the sweep")
	byId := idsOf(pools)
	for _, o := range []*Order{good, unsorted, docked} {
		require.Contains(t, byId, poolId(testRouter, o.Maker, o.Hash()))
	}
	require.Len(t, pools, 3)
	got := func(r DiscardReason) float64 {
		return recorder.get(metricDiscard + "|" + string(r) + "@portal")
	}
	require.Equal(t, float64(2), got(ReasonApp), "the foreign app and the launch app with no launch router configured")
	require.Equal(t, float64(3), got(ReasonTokenCount), "3 tokens, 1 token, the same token twice")
	require.Equal(t, float64(1), got(ReasonTokenUnprobed))
	require.Equal(t, float64(1), got(ReasonUnsupportedOpcode))
	require.Equal(t, float64(1), got(ReasonTxOriginGuard))
	require.Equal(t, float64(1), got(ReasonMakerMismatch))
	require.Equal(t, float64(2), got(ReasonNonCanonical), "a hash that does not match the bytes, and bytes that are no order")
	require.Zero(t, recorder.get(metricDiscard+"|"+string(ReasonApp)+"@logs"), "no log source label")

	// Tokens leave in ascending order whatever order the Portal sent.
	p := byId[poolId(testRouter, unsorted.Maker, unsorted.Hash())]
	require.Equal(t, []string{tokenA, tokenB}, []string{p.Tokens[0].Address, p.Tokens[1].Address})

	// With the launch router configured the launch strategy is admitted, flagged launch.
	u2, _ := newPortalLister(t, n, fp, portalConfig(fp, func(c *Config) { c.PortalPageLimit = 100; c.LaunchRouterAddress = testLaunchHex }))
	pools, _, err = u2.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	lp, ok := idsOf(pools)[poolId(testLaunch, launchOrder.Maker, launchOrder.Hash())]
	require.True(t, ok)
	require.True(t, decodeStatic(t, lp).Launch)
	require.Len(t, pools, 4)
}

const metricDiscard = "discarded_strategy_count"

// The Portal can list a strategy that is docked: the pool is created and the tracker reads tokensCount 0xff in
// its first batch, so it is docked with no liquidity and never quotes.
func TestPortalDockedLagItem(t *testing.T) {
	o := orderOf(testMaker, prog(iFlat(1), iSwap()))
	fp := newFakePortal(t, portalItemOf(1, o, testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)

	chain := newFakeChain()
	chain.setStrategy(poolOpts{}, pools[0], tokensCountDocked, e18(0), e18(0))
	chain.setWallet(testMaker.Hex(), tokenA, e18(900), unlimited)
	chain.setWallet(testMaker.Hex(), tokenB, e18(900), unlimited)
	tr, _ := chain.tracker(t)
	out, errs := tr.GetNewPoolStates(context.Background(), pools)
	require.NoError(t, errs[0])
	x := decodeExtra(t, out[0])
	require.True(t, x.Docked)
	require.Equal(t, entity.PoolReserves{"0", "0"}, out[0].Reserves)
	require.ErrorIs(t, quotes(t, out[0]), ErrStrategyTerminal)
}

// The Portal and the log path make byte-identical pools for the same strategies: the golden vectors of live
// strategies (both routers, both token orders), listed once by each path.
func TestPortalAndLogsMakeIdenticalPools(t *testing.T) {
	recorder.reset()
	vectors := loadGolden(t)
	nodeLogs := newFakeNode(t, 1000)
	probed := map[string]string{}
	seen := map[string]bool{}
	var items []map[string]any
	for i, v := range vectors {
		if seen[v.Strategy] {
			continue
		}
		seen[v.Strategy] = true
		for _, tk := range v.Tokens {
			probed[strings.ToLower(tk)] = ProbeOK
		}
		o, err := DecodeOrder(hexutil.MustDecode(v.Strategy))
		require.NoError(t, err)
		app := common.HexToAddress(v.App)
		// The log path lists the tokens in Pushed order (the golden order, which is sometimes descending);
		// the Portal sorts them ascending.
		ship(t, nodeLogs, 10, txHash(2000+i), 0, o, app, strings.ToLower(v.Tokens[0]), strings.ToLower(v.Tokens[1]))
		sorted := []string{strings.ToLower(v.Tokens[0]), strings.ToLower(v.Tokens[1])}
		if sorted[0] > sorted[1] {
			sorted[0], sorted[1] = sorted[1], sorted[0]
		}
		items = append(items, portalItemOf(1, o, app, sorted...))
	}
	fp := newFakePortal(t, items...)
	mut := func(c *Config) { c.ProbedTokens = probed; c.LaunchRouterAddress = testLaunchHex; c.PortalPageLimit = 7 }

	logLister := newLister(t, nodeLogs, Config{DexId: DexType, ChainId: 1, StartBlock: 1, ProbedTokens: probed, LaunchRouterAddress: testLaunchHex})
	logPools, _, err := logLister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)

	nodePortal := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, nodePortal, fp, portalConfig(fp, mut))
	portalPools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, nodePortal.requestLog())

	require.Len(t, logPools, len(seen))
	require.Len(t, portalPools, len(seen))
	lm, pm := idsOf(logPools), idsOf(portalPools)
	for id, lp := range lm {
		pp, ok := pm[id]
		require.True(t, ok, "pool %s is listed by the Portal too", id)
		lp.Timestamp, pp.Timestamp = 0, 0 // the listing time is the only field that differs
		require.Equal(t, lp, pp, "pool %s", id)
		lb, _ := json.Marshal(lp)
		pb, _ := json.Marshal(pp)
		require.Equal(t, string(lb), string(pb))
	}
}

// The key appears in no error, log line, metric, or printed settings, whatever the Portal answers, including a
// Portal that echoes the Authorization header and the key back in its body.
func TestPortalKeyNeverLeaks(t *testing.T) {
	recorder.reset()
	echo := func(w http.ResponseWriter, r *http.Request, code int) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte("auth=" + r.Header.Get("Authorization") + " key=" + testPortalKey))
	}
	scenarios := map[string]func(call int, w http.ResponseWriter, r *http.Request) bool{
		"401 echo": func(_ int, w http.ResponseWriter, r *http.Request) bool { echo(w, r, 401); return true },
		"429 echo": func(_ int, w http.ResponseWriter, r *http.Request) bool { echo(w, r, 429); return true },
		"500 echo": func(_ int, w http.ResponseWriter, r *http.Request) bool { echo(w, r, 500); return true },
		"200 echo": func(_ int, w http.ResponseWriter, r *http.Request) bool { echo(w, r, 200); return true },
		"418 echo": func(_ int, w http.ResponseWriter, r *http.Request) bool { echo(w, r, 418); return true },
		"redirect key": func(_ int, w http.ResponseWriter, r *http.Request) bool {
			http.Redirect(w, r, "https://example.invalid/?k="+testPortalKey, http.StatusFound)
			return true
		},
		"hijack": func(_ int, w http.ResponseWriter, _ *http.Request) bool {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}
			return true
		},
	}
	for name, fault := range scenarios {
		fp := newFakePortal(t, portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB))
		fp.fault = fault
		n := newFakeNode(t, 1000)
		u, capture := newPortalLister(t, n, fp, portalConfig(fp))
		_, perr := u.portal.sweep(context.Background())
		require.NotNil(t, perr, name)
		require.NotContains(t, perr.Error(), testPortalKey, name)
		require.NotContains(t, perr.Error(), "Bearer", name)
		_, _, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
		require.NoError(t, err, name)
		require.NotContains(t, capture.joined(), testPortalKey, name)
		require.NotContains(t, capture.joined(), "Bearer", name)
		for _, printed := range []string{fmt.Sprintf("%v %+v %#v", u.cfg, u.cfg, u.cfg), fmt.Sprintf("%v %+v %#v", u.cfg.portal, u.cfg.portal, u.cfg.portal),
			fmt.Sprintf("%v %+v", u.portal.cfg, u.portal.cfg)} {
			require.NotContains(t, printed, testPortalKey, name)
		}
		// The client's own error text, scrubbed, hides a key that a transport put in it.
		require.NotContains(t, u.cfg.portal.scrub("dial https://x/?k="+testPortalKey+" failed"), testPortalKey)
	}
	require.NotContains(t, fmt.Sprint(recorder.counts), testPortalKey)
	b, err := json.Marshal(u2secret())
	require.NoError(t, err)
	require.NotContains(t, string(b), testPortalKey)
}

func u2secret() any { return struct{ K *secret }{&secret{testPortalKey}} }

// A redirect is refused: the key is never sent to the target.
func TestPortalRefusesRedirect(t *testing.T) {
	recorder.reset()
	var mu sync.Mutex
	var seen []string
	target := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
	}))
	defer target.Close()
	fp := newFakePortal(t, portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB))
	fp.fault = func(_ int, w http.ResponseWriter, r *http.Request) bool {
		http.Redirect(w, r, target.URL+"/x", http.StatusTemporaryRedirect)
		return true
	}
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	_, _, err := u.GetNewPools(context.Background(), metaJSON(t, 5, 0))
	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	require.Empty(t, seen, "the redirect target was never called")
	require.Equal(t, float64(1), recorder.get(metricPortalFallback+"|"+portalNetwork))
}

// A client built by the lister has no redirect, a bearer header and the base URL.
func TestPortalClientDefaults(t *testing.T) {
	fp := newFakePortal(t)
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	require.IsType(t, &resty.Client{}, u.portal.http)
	require.Equal(t, fp.URL, u.portal.http.BaseURL)
	_, perr := u.portal.sweep(context.Background())
	require.Nil(t, perr, "an empty list is a full sweep")
	require.Equal(t, 1, fp.callCount())
}

// An item with an unreadable token address is a bad item (the sweep falls back), the items around it are still
// read, and the bad ones are counted.
func TestPortalShippedBadTokenAddress(t *testing.T) {
	fp := newFakePortal(t)
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	orders := testOrders(3)
	bad1 := portalItemOf(1, orders[0], testRouter, tokenA, "0xnothex")
	bad2 := portalItemOf(1, orders[1], testRouter, "0xnothex", "0xalsonot", tokenB)
	badMaker := portalItemOf(1, orders[0], testRouter, tokenA, tokenB)
	badMaker["maker"] = "0xnothex"
	raw, err := json.Marshal([]any{badMaker, bad1, bad2, portalItemOf(1, orders[2], testRouter, tokenA, tokenB, tokenA)})
	require.NoError(t, err)
	var items []portalItem
	require.NoError(t, json.Unmarshal(raw, &items))
	ships, bad := u.portalShipped(items)
	require.Equal(t, 3, bad)
	require.Len(t, ships, 1, "the good item after the bad ones is read")
	require.Len(t, ships[0].tokens, 2, "a repeated token is one token")
}

// With no stored cursor the log scan starts at startBlock; a stored cursor wins; and the limit of the unprobed
// report cap is accepted while one above it is not.
func TestListerStartAndReportCap(t *testing.T) {
	n := newFakeNode(t, 1000)
	u := newLister(t, n, Config{StartBlock: 100, LogWindow: 50, MaxWindowsPerRun: 1})
	_, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, [][2]uint64{{100, 149}}, n.requestLog())
	require.Equal(t, uint64(149), cursorOf(t, meta))
	require.Equal(t, uint64(99), u.startCursor(Metadata{}))
	require.Equal(t, uint64(7), u.startCursor(Metadata{LastScannedBlock: 7}))

	for limit, want := range map[int]string{maxUnprobedReportsLimit: "", maxUnprobedReportsLimit + 1: "above"} {
		cfg := testConfig(func(c *Config) { c.MaxUnprobedReportsPerRun = limit })
		_, err := cfg.settings()
		if want == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, want)
		}
	}
}

// A strategy shipped at or before H that the Portal has not indexed yet is not lost when the Portal then goes
// down: a full sweep moves the cursor only to the previous sweep's head, so the log fallback re-scans the
// blocks since then and finds it. (Moving the cursor to H, as before, skipped it for good.)
func TestPortalLagItemRecoveredByFallback(t *testing.T) {
	recorder.reset()
	orders := testOrders(3)
	fp := newFakePortal(t, portalItemOf(1, orders[0], testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp, func(c *Config) { c.LogWindow = 500 }))
	setHead := func(h uint64) {
		n.mu.Lock()
		defer n.mu.Unlock()
		n.head = h
	}

	// Sweep 1 at head 1000 (H 988): a first sweep, only the head is stored.
	_, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)

	// A strategy ships at block 1050 and is on chain, but the Portal does not list it yet. Sweep 2 at head
	// 1100 (H 1088) lists one pool and moves the cursor to 988, not past the lagging strategy.
	lagging := orders[1]
	ship(t, n, 1050, txHash(1), 0, lagging, testRouter, tokenA, tokenB)
	setHead(1100)
	pools, meta, err := u.GetNewPools(context.Background(), meta)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, Metadata{988, 1088}, metaOf(t, meta))
	require.Empty(t, n.requestLog())

	// Sweep 3 at head 1200: the Portal is down. The log scan starts at 989 and lists the lagging strategy.
	fp.mu.Lock()
	fp.fault = statusFault(503)
	fp.mu.Unlock()
	setHead(1200)
	pools, meta, err = u.GetNewPools(context.Background(), meta)
	require.NoError(t, err)
	require.Equal(t, float64(1), recorder.get(metricPortalFallback+"|"+portalServer))
	require.Equal(t, [][2]uint64{{989, 1188}}, n.requestLog())
	require.Contains(t, idsOf(pools), poolId(testRouter, lagging.Maker, lagging.Hash()))
	require.Equal(t, Metadata{1188, 1088}, metaOf(t, meta), "the cursor is the scan's, the stored sweep head stays")
}

// Over three full sweeps the cursor trails the head by one sweep and the stored head follows it.
func TestPortalCursorOverThreeSweeps(t *testing.T) {
	fp := newFakePortal(t, portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	var meta []byte
	for i, want := range []Metadata{{0, 988}, {988, 1088}, {1088, 1188}} {
		n.mu.Lock()
		n.head = 1000 + 100*uint64(i)
		n.mu.Unlock()
		_, next, err := u.GetNewPools(context.Background(), meta)
		require.NoError(t, err)
		require.Equal(t, want, metaOf(t, next), "sweep %d", i+1)
		meta = next
	}
	require.Empty(t, n.requestLog(), "a full sweep reads no logs")
}

// The metadata round trip, and metadata stored before LastPortalHead existed.
func TestListerMetadataCompatibility(t *testing.T) {
	old := []byte(`{"lastScannedBlock":7}`)
	require.Equal(t, Metadata{LastScannedBlock: 7}, metaOf(t, old))
	again, err := json.Marshal(Metadata{LastScannedBlock: 7})
	require.NoError(t, err)
	require.JSONEq(t, string(old), string(again), "a zero head is not written: logs-only listers keep the old form")
	require.NotContains(t, string(again), "lastPortalHead")
	both, err := json.Marshal(Metadata{LastScannedBlock: 7, LastPortalHead: 9})
	require.NoError(t, err)
	require.JSONEq(t, `{"lastScannedBlock":7,"lastPortalHead":9}`, string(both))
	require.Equal(t, Metadata{7, 9}, metaOf(t, both))

	// Old metadata at a full sweep: the cursor stays and the head is stored.
	fp := newFakePortal(t, portalItemOf(1, testOrders(1)[0], testRouter, tokenA, tokenB))
	n := newFakeNode(t, 1000)
	u, _ := newPortalLister(t, n, fp, portalConfig(fp))
	_, meta, err := u.GetNewPools(context.Background(), old)
	require.NoError(t, err)
	require.Equal(t, Metadata{7, 988}, metaOf(t, meta))

	// A log-only lister never writes the field, and a log run keeps the one a sweep stored.
	logOnly := newLister(t, newFakeNode(t, 1000), Config{StartBlock: 100, LogWindow: 50, MaxWindowsPerRun: 1})
	_, meta, err = logOnly.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotContains(t, string(meta), "lastPortalHead")
	fp.mu.Lock()
	fp.fault = statusFault(500)
	fp.mu.Unlock()
	_, meta, err = u.GetNewPools(context.Background(), metaJSON(t, 5, 900))
	require.NoError(t, err)
	require.Equal(t, Metadata{55, 900}, metaOf(t, meta))
}
