package nemoprop

import (
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

var (
	testProxy = common.HexToAddress("0x00000000000000000000000000000000000e0e01")
	testOther = common.HexToAddress("0x00000000000000000000000000000000000e0e02")
	testUSDC  = common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	testWETH  = common.HexToAddress("0x4200000000000000000000000000000000000006")
	testCBBTC = common.HexToAddress("0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf")
)

func hexAddr(a common.Address) string { return hexutil.Encode(a[:]) }

// ---------------------------------------------------------------------------
// Fake live feed

type fakeFeed struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
	token    string
	conns    []*websocket.Conn
	subs     []feedSubscribe
	authSeen []string

	accepted atomic.Int32
	rejected atomic.Int32
}

func newFakeFeed(t *testing.T, token string) *fakeFeed {
	f := &fakeFeed{t: t, token: token}
	upgrader := websocket.Upgrader{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		f.mu.Lock()
		f.authSeen = append(f.authSeen, auth)
		ok := auth == "Bearer "+f.token
		f.mu.Unlock()
		if !ok || strings.Contains(r.URL.RawQuery, "token") {
			f.rejected.Add(1)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		var sub feedSubscribe
		if _, data, err := conn.ReadMessage(); err != nil || json.Unmarshal(data, &sub) != nil {
			_ = conn.Close()
			return
		}
		f.mu.Lock()
		f.conns = append(f.conns, conn)
		f.subs = append(f.subs, sub)
		f.mu.Unlock()
		f.accepted.Add(1)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(func() {
		if v, ok := feeds.LoadAndDelete(feedKey(f.url(), 8453, hexAddr(testProxy))); ok {
			v.(*feedClient).close()
		}
		f.dropAll()
		f.srv.Close()
	})
	return f
}

func (f *fakeFeed) url() string { return "ws" + strings.TrimPrefix(f.srv.URL, "http") }

// config is a tracker config for this feed with long freshness windows, so
// only a test's own changes stop quoting.
func (f *fakeFeed) config() *Config {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &Config{DexID: DexType, ChainID: 8453, Address: hexAddr(testProxy),
		Feed: FeedConfig{URL: f.url(), AuthToken: f.token, FreshMs: 60_000, MaxAgeMs: 60_000}}
}

func (f *fakeFeed) setToken(token string) {
	f.mu.Lock()
	f.token = token
	f.mu.Unlock()
}

func (f *fakeFeed) waitConns(n int32) {
	f.t.Helper()
	require.Eventually(f.t, func() bool { return f.accepted.Load() >= n }, 5*time.Second, 5*time.Millisecond,
		"feed connection %d never arrived", n)
}

// send writes a raw frame to the newest connection.
func (f *fakeFeed) send(frame []byte) {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	require.NotEmpty(f.t, f.conns)
	require.NoError(f.t, f.conns[len(f.conns)-1].WriteMessage(websocket.TextMessage, frame))
}

func (f *fakeFeed) push(msg feedMessage) {
	f.t.Helper()
	data, err := json.Marshal(msg)
	require.NoError(f.t, err)
	f.send(data)
}

func (f *fakeFeed) dropAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.conns {
		_ = c.Close()
	}
	f.conns = nil
}

// linearLadders are straight lines through the origin, so the spline is
// exact everywhere: out = rate * in.
func linearLadders(rate0, rate1 float64) [2][]ladder.Point {
	return [2][]ladder.Point{
		{{1000, 1000 * rate0}, {2000, 2000 * rate0}, {3000, 3000 * rate0}},
		{{1000, 1000 * rate1}, {2000, 2000 * rate1}, {3000, 3000 * rate1}},
	}
}

func snapshotFrame(seq uint64, rate0, rate1 float64) feedMessage {
	return feedMessage{
		V: feedProtocolVersion, Type: feedMsgSnapshot, ChainID: 8453, Proxy: hexAddr(testProxy),
		Seq: seq, TimestampMs: time.Now().UnixMilli(), BlockNumber: 1000 + seq,
		Base: hexAddr(testUSDC),
		Tokens: map[string]feedToken{
			hexAddr(testUSDC): {Balance: "1000000000", Allowance: "1000000000"},
			hexAddr(testWETH): {Balance: "1000000000", Allowance: "1000000000"},
		},
		Markets: map[string]feedMarket{hexAddr(testWETH): {Ladders: linearLadders(rate0, rate1)}},
	}
}

// testPool is a freshly discovered (USDC, WETH) pool.
func testPool(t *testing.T) entity.Pool {
	t.Helper()
	staticExtra, err := json.Marshal(StaticExtra{Address: hexAddr(testProxy)})
	require.NoError(t, err)
	return entity.Pool{
		Address:  poolAddress(testProxy, testUSDC, testWETH),
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexAddr(testUSDC), Decimals: 6, Swappable: true},
			{Address: hexAddr(testWETH), Decimals: 18, Swappable: true},
		},
		Extra:       "{}",
		StaticExtra: string(staticExtra),
	}
}

// btcPool is the (USDC, cbBTC) pool of the same proxy.
func btcPool(t *testing.T) entity.Pool {
	p := testPool(t)
	p.Address = poolAddress(testProxy, testUSDC, testCBBTC)
	p.Tokens[1] = &entity.PoolToken{Address: hexAddr(testCBBTC), Decimals: 8, Swappable: true}
	return p
}

// pooledState is a pool as the tracker writes it from a snapshot received
// age ago, with fresh window freshMs, decay decayBps and max age maxAgeMs.
func pooledState(t *testing.T, ladders [2][]ladder.Point, reserves entity.PoolReserves, age time.Duration,
	freshMs, maxAgeMs, decayBps int64) entity.Pool {
	t.Helper()
	p := testPool(t)
	extra, err := json.Marshal(Extra{Extra: ladder.Extra{Ladders: ladders},
		ReceivedAtMs: time.Now().Add(-age).UnixMilli(), FreshMs: freshMs, MaxAgeMs: maxAgeMs, DecayBps: decayBps})
	require.NoError(t, err)
	p.Extra, p.Reserves, p.BlockNumber = string(extra), reserves, 1001
	return p
}

// ---------------------------------------------------------------------------
// Fake chain behind a JSON-RPC server, serving the proxy's getMarkets_v1.

type fakeChain struct {
	mu      sync.Mutex
	markets []common.Address
	units   map[common.Address]*big.Int
}

func newFakeChain() *fakeChain {
	return &fakeChain{
		markets: []common.Address{testWETH, testCBBTC},
		units:   map[common.Address]*big.Int{testWETH: big.NewInt(1e18), testCBBTC: big.NewInt(1e8)},
	}
}

func (c *fakeChain) rpcClient(t *testing.T) *ethrpc.Client {
	srv := httptest.NewServer(http.HandlerFunc(c.serveHTTP))
	t.Cleanup(srv.Close)
	return ethrpc.New(srv.URL)
}

func (c *fakeChain) marketsBytes() []byte {
	var out []byte
	for _, m := range c.markets {
		word := make([]byte, 32)
		copy(word, m[:])
		c.units[m].FillBytes(word[20:])
		out = append(out, word...)
	}
	return out
}

func (c *fakeChain) serveHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(body, &req)
	var msg struct {
		To    common.Address `json:"to"`
		Data  hexutil.Bytes  `json:"data"`
		Input hexutil.Bytes  `json:"input"`
	}
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params[0], &msg)
	}
	data := append(msg.Data, msg.Input...)

	resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
	method, err := nemoSwapABI.MethodById(data)
	if req.Method != "eth_call" || msg.To != testProxy || err != nil || method.Name != "getMarkets_v1" {
		resp["error"] = map[string]any{"code": 3, "message": "execution reverted"}
	} else {
		c.mu.Lock()
		out, _ := method.Outputs.Pack(testUSDC, c.marketsBytes())
		c.mu.Unlock()
		resp["result"] = hexutil.Encode(out)
	}
	_ = json.NewEncoder(w).Encode(resp)
}
