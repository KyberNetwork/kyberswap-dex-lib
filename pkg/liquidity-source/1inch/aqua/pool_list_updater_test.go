package oneinchaqua

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
)

// The lister tests are not parallel: they share the process-wide metric hook.
type metricRecorder struct {
	mu     sync.Mutex
	counts map[string]float64
}

var recorder = &metricRecorder{counts: map[string]float64{}}

func init() {
	metrics.SetIncrFn(func(_ context.Context, name string, tags map[string]string, value float64) {
		key := name
		for _, k := range []string{"reason", "token"} {
			if v, ok := tags[k]; ok {
				key += "|" + v
			}
		}
		recorder.mu.Lock()
		recorder.counts[key] += value
		if src, ok := tags["source"]; ok {
			recorder.counts[key+"@"+src] += value
		}
		recorder.mu.Unlock()
	})
}

func (r *metricRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts = map[string]float64{}
}

func (r *metricRecorder) get(key string) float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.counts[key]
}

// fakeNode serves eth_blockNumber and eth_getLogs.
type fakeNode struct {
	*httptest.Server
	mu        sync.Mutex
	head      uint64
	logs      []types.Log
	maxSpan   uint64 // a wider range answers with a limit error, or with spanFault when set
	spanFault *fault
	badBlock  map[uint64]bool // a range holding one of these blocks answers "invalid block range"
	failFrom  uint64          // a range reaching this block or above answers a non-range error
	failBlock uint64          // a range (within maxSpan) holding this block answers fault
	fault     fault           // what the failFrom and failBlock ranges answer; the zero value is a plain error
	requests  [][2]uint64
}

// fault is a failure the node answers a log range with.
type fault struct {
	message string // JSON-RPC error message
	status  int    // HTTP status instead of a JSON-RPC reply
	drop    bool   // close the connection with no reply
}

func (n *fakeNode) answerFault(w http.ResponseWriter, reply func(any, string)) {
	answerFault(n.fault, w, reply)
}

func answerFault(f fault, w http.ResponseWriter, reply func(any, string)) {
	switch {
	case f.drop:
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	case f.status != 0:
		http.Error(w, http.StatusText(f.status), f.status)
	case f.message != "":
		reply(nil, f.message)
	default:
		reply(nil, "upstream exploded")
	}
}

func newFakeNode(t *testing.T, head uint64) *fakeNode {
	n := &fakeNode{head: head, badBlock: map[uint64]bool{}}
	n.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Id     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		reply := func(result any, errMsg string) {
			m := map[string]any{"jsonrpc": "2.0", "id": req.Id}
			if errMsg != "" {
				m["error"] = map[string]any{"code": -32000, "message": errMsg}
			} else {
				m["result"] = result
			}
			_ = json.NewEncoder(w).Encode(m)
		}
		n.mu.Lock()
		defer n.mu.Unlock()
		switch req.Method {
		case "eth_blockNumber":
			reply(hexutil.EncodeUint64(n.head), "")
		case "eth_getLogs":
			var q struct {
				FromBlock string `json:"fromBlock"`
				ToBlock   string `json:"toBlock"`
			}
			_ = json.Unmarshal(req.Params[0], &q)
			from, _ := hexutil.DecodeUint64(q.FromBlock)
			to, _ := hexutil.DecodeUint64(q.ToBlock)
			n.requests = append(n.requests, [2]uint64{from, to})
			if n.failFrom > 0 && to >= n.failFrom {
				n.answerFault(w, reply)
				return
			}
			if n.maxSpan > 0 && to-from+1 > n.maxSpan {
				if n.spanFault != nil {
					answerFault(*n.spanFault, w, reply)
					return
				}
				reply(nil, "query exceeds limit of results")
				return
			}
			if n.failBlock > 0 && n.failBlock >= from && n.failBlock <= to {
				n.answerFault(w, reply)
				return
			}
			for b := range n.badBlock {
				if b >= from && b <= to {
					reply(nil, "invalid block range")
					return
				}
			}
			var out []types.Log
			for _, lg := range n.logs {
				if lg.BlockNumber >= from && lg.BlockNumber <= to {
					out = append(out, lg)
				}
			}
			if out == nil {
				out = []types.Log{}
			}
			reply(out, "")
		default:
			reply(nil, "execution reverted")
		}
	}))
	t.Cleanup(n.Close)
	return n
}

func (n *fakeNode) client() *ethrpc.Client { return ethrpc.New(n.URL) }

func (n *fakeNode) requestLog() [][2]uint64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([][2]uint64(nil), n.requests...)
}

func (n *fakeNode) add(logs ...types.Log) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.logs = append(n.logs, logs...)
}

var aquaAddr = testAqua

func mkLog(event string, block uint64, index uint, tx common.Hash, data []byte) types.Log {
	return types.Log{Address: aquaAddr, Topics: []common.Hash{aquaABI.Events[event].ID}, Data: data,
		BlockNumber: block, TxHash: tx, Index: index}
}

func shippedLog(t testing.TB, block uint64, index uint, tx common.Hash, maker, app common.Address, hash common.Hash, strategy []byte) types.Log {
	t.Helper()
	data, err := shippedEvent.Inputs.Pack(maker, app, [32]byte(hash), strategy)
	require.NoError(t, err)
	return mkLog("Shipped", block, index, tx, data)
}

func pushedLog(t testing.TB, block uint64, index uint, tx common.Hash, maker, app common.Address, hash common.Hash, token string) types.Log {
	t.Helper()
	data, err := pushedEvent.Inputs.Pack(maker, app, [32]byte(hash), common.HexToAddress(token), big.NewInt(1000))
	require.NoError(t, err)
	return mkLog("Pushed", block, index, tx, data)
}

func dockedLog(t testing.TB, block uint64, index uint, tx common.Hash, maker, app common.Address, hash common.Hash) types.Log {
	t.Helper()
	data, err := dockedEvent.Inputs.Pack(maker, app, [32]byte(hash))
	require.NoError(t, err)
	return mkLog("Docked", block, index, tx, data)
}

var (
	tokenA = "0x1111111111111111111111111111111111111111"
	tokenB = "0x2222222222222222222222222222222222222222"
	tokenC = "0x3333333333333333333333333333333333333333"
)

func probedAll(tokens ...string) map[string]string {
	m := map[string]string{}
	for _, tk := range tokens {
		m[tk] = ProbeOK
	}
	return m
}

func txHash(i int) common.Hash { return common.BigToHash(big.NewInt(int64(i))) }

// ship adds a Shipped log and Pushed logs for the tokens, the way Aqua.ship emits them.
func ship(t testing.TB, n *fakeNode, block uint64, tx common.Hash, firstIndex uint, o *Order, app common.Address, tokens ...string) common.Hash {
	t.Helper()
	hash := o.Hash()
	logs := []types.Log{shippedLog(t, block, firstIndex, tx, o.Maker, app, hash, o.Encode())}
	for i, tk := range tokens {
		logs = append(logs, pushedLog(t, block, firstIndex+1+uint(i), tx, o.Maker, app, hash, tk))
	}
	n.add(logs...)
	return hash
}

func newLister(t *testing.T, n *fakeNode, cfg Config) *PoolsListUpdater {
	t.Helper()
	if cfg.ProbedTokens == nil {
		cfg.ProbedTokens = probedAll(tokenA, tokenB, tokenC)
	}
	base := testConfig()
	if cfg.Confirmations == nil {
		cfg.Confirmations = base.Confirmations
	}
	cfg.AquaAddress, cfg.RouterAddress = base.AquaAddress, base.RouterAddress
	if cfg.StartBlock == 0 {
		cfg.StartBlock = 1
	}
	if cfg.LogWindow == 0 {
		cfg.LogWindow = base.LogWindow
	}
	if cfg.MaxWindowsPerRun == 0 {
		cfg.MaxWindowsPerRun = base.MaxWindowsPerRun
	}
	u, err := NewPoolsListUpdater(&cfg, n.client())
	require.NoError(t, err)
	return u
}

func TestListerJoinAndPoolFields(t *testing.T) {
	n := newFakeNode(t, 1000)
	m2 := common.HexToAddress("0x9999999999999999999999999999999999999999")
	tx := txHash(1)
	o1, o2 := orderOf(testMaker, prog(iFlat(1), iSwap())), orderOf(m2, prog(iFlat(2), iSwap()))
	// Two strategies shipped in one tx, plus noise a join on the tx alone would pick up: a Pushed of the
	// first strategy that precedes its Shipped log, a Pushed of another strategy, a repeat push of a
	// token of the first one.
	n.add(pushedLog(t, 10, 0, tx, testMaker, testRouter, o1.Hash(), tokenC))
	h1 := ship(t, n, 10, tx, 1, o1, testRouter, tokenA, tokenB)
	h2 := ship(t, n, 10, tx, 4, o2, testRouter, tokenB, tokenC)
	n.add(pushedLog(t, 10, 7, tx, testMaker, testRouter, h1, tokenA))

	u := newLister(t, n, Config{DexId: "1inch-aqua", StartBlock: 1})
	pools, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	byId := map[string]entity.Pool{}
	for _, p := range pools {
		byId[p.Address] = p
	}
	p1, p2 := byId[poolId(testRouter, testMaker, h1)], byId[poolId(testRouter, m2, h2)]
	require.NotEmpty(t, p1.Address)
	require.NotEmpty(t, p2.Address)
	require.Regexp(t, "^0x[0-9a-f]{64}$", p1.Address)

	require.Equal(t, DexType, p1.Type)
	require.Equal(t, "1inch-aqua", p1.Exchange)
	require.Equal(t, entity.PoolReserves{"0", "0"}, p1.Reserves)
	require.Equal(t, []string{tokenA, tokenB}, []string{p1.Tokens[0].Address, p1.Tokens[1].Address}, "ascending address order")
	require.True(t, p1.Tokens[0].Swappable && p1.Tokens[1].Swappable)
	require.Equal(t, []string{tokenB, tokenC}, []string{p2.Tokens[0].Address, p2.Tokens[1].Address})
	require.Equal(t, uint64(0), p1.BlockNumber)

	static := decodeStatic(t, p1)
	require.Equal(t, addrHex(testMaker), static.Maker)
	require.Equal(t, addrHex(testRouter), static.App)
	require.Equal(t, h1.Hex(), static.StrategyHash)
	back, err := static.order()
	require.NoError(t, err)
	require.Equal(t, h1, back.Hash(), "the static extra rebuilds the order the router hashes")

	// What the lister writes the simulator reads, and it cannot quote before the first refresh.
	sim, err := NewPoolSimulator(p1)
	require.NoError(t, err)
	_, err = quoteOut(sim, 0, e18(1), nil)
	require.ErrorIs(t, err, ErrNoWalletSnapshot)
	require.Nil(t, sim.CalculateLimit())

	var m Metadata
	require.NoError(t, json.Unmarshal(meta, &m))
	require.Equal(t, uint64(1000), m.LastScannedBlock)
}

func TestListerDiscardReasons(t *testing.T) {
	recorder.reset()
	n := newFakeNode(t, 1000)
	launchOnly := testLaunch
	foreignApp := common.HexToAddress("0xf3da3145b208ebfa94fad073ba9c03d6e8746ffe")
	other := common.HexToAddress("0x8888888888888888888888888888888888888888")
	good := prog(iFlat(1), iSwap())
	i := 0
	next := func() (common.Hash, uint) { i++; return txHash(100 + i), 0 }

	// wanted discard reasons, one strategy each.
	want := map[DiscardReason]int{}
	add := func(reason DiscardReason, o *Order, app common.Address, tokens ...string) {
		tx, idx := next()
		ship(t, n, 20, tx, idx, o, app, tokens...)
		if reason != "" {
			want[reason]++
		}
	}
	add(ReasonApp, orderOf(testMaker, good), foreignApp, tokenA, tokenB)
	add(ReasonApp, orderOf(testMaker, good), launchOnly, tokenA, tokenB) // the chain has no launch router configured
	add(ReasonTxOriginGuard, orderOf(testMaker, prog(guard(), iFlat(1), iSwap())), testRouter, tokenA, tokenB)
	add(ReasonUnsupportedOpcode, orderOf(testMaker, prog(ins(0x20, 1, 2), iSwap())), testRouter, tokenA, tokenB)
	add(ReasonTokenCount, orderOf(testMaker, good), testRouter, tokenA)
	add(ReasonTokenCount, orderOf(testMaker, good), testRouter, tokenA, tokenB, tokenC)
	add(ReasonTokenUnprobed, orderOf(testMaker, prog(iFlat(3), iSwap())), testRouter, tokenA, "0x4444444444444444444444444444444444444444")
	add(ReasonTokenExcluded, orderOf(testMaker, prog(iFlat(4), iSwap())), testRouter, tokenA, "0x5555555555555555555555555555555555555555")
	add("", orderOf(testMaker, prog(iFlat(5), iSwap())), testRouter, tokenA, tokenB) // admitted
	add("", orderOf(testMaker, prog(iFlat(6), iSwap())), testRouter, tokenA, "0x6666666666666666666666666666666666666666")
	// A maker that is not the order's maker: shipped by `other`, order names testMaker.
	tx, idx := next()
	o := orderOf(testMaker, prog(iFlat(7), iSwap()))
	n.add(shippedLog(t, 20, idx, tx, other, testRouter, o.Hash(), o.Encode()),
		pushedLog(t, 20, idx+1, tx, other, testRouter, o.Hash(), tokenA), pushedLog(t, 20, idx+2, tx, other, testRouter, o.Hash(), tokenB))
	want[ReasonMakerMismatch]++
	// Non-canonical bytes: the hash Aqua keyed is over the raw bytes.
	tx, idx = next()
	raw := append(orderOf(testMaker, good).Encode(), make([]byte, 32)...)
	hash := crypto.Keccak256Hash(raw)
	n.add(shippedLog(t, 20, idx, tx, testMaker, testRouter, hash, raw),
		pushedLog(t, 20, idx+1, tx, testMaker, testRouter, hash, tokenA), pushedLog(t, 20, idx+2, tx, testMaker, testRouter, hash, tokenB))
	want[ReasonNonCanonical]++
	// A hash that is not the order's hash.
	tx, idx = next()
	o = orderOf(testMaker, prog(iFlat(8), iSwap()))
	n.add(shippedLog(t, 20, idx, tx, testMaker, testRouter, common.HexToHash("0x1234"), o.Encode()),
		pushedLog(t, 20, idx+1, tx, testMaker, testRouter, common.HexToHash("0x1234"), tokenA), pushedLog(t, 20, idx+2, tx, testMaker, testRouter, common.HexToHash("0x1234"), tokenB))
	want[ReasonNonCanonical]++
	// Docked in the scanned window.
	tx, idx = next()
	od := orderOf(testMaker, prog(iFlat(9), iSwap()))
	hd := ship(t, n, 21, tx, idx, od, testRouter, tokenA, tokenB)
	n.add(dockedLog(t, 25, 0, txHash(900), testMaker, testRouter, hd))
	want[ReasonDockedInWindow]++
	// Docked in the window with a token that is not probed: docked, not unprobed.
	tx, idx = next()
	odu := orderOf(testMaker, prog(iFlat(11), iSwap()))
	hdu := ship(t, n, 21, tx, idx, odu, testRouter, tokenA, "0x4444444444444444444444444444444444444444")
	n.add(dockedLog(t, 26, 0, txHash(901), testMaker, testRouter, hdu))
	want[ReasonDockedInWindow]++
	// Launch router configured: zero-transfer-reverting token with a protocol fee is out, without a fee it is in.
	tx, idx = next()
	ship(t, n, 22, tx, idx, orderOf(testMaker, prog(iProtocol(5, testRecipient), iSwap())), launchOnly, tokenA, "0x7777777777777777777777777777777777777777")
	want[ReasonZeroTransferRevertsFees]++

	probed := probedAll(tokenA, tokenB, tokenC)
	probed["0x5555555555555555555555555555555555555555"] = ProbeExcluded
	probed["0x6666666666666666666666666666666666666666"] = ProbeZeroTransferReverts
	probed["0x7777777777777777777777777777777777777777"] = ProbeZeroTransferReverts
	zero := uint64(0)

	// Without a launch router configured the launch-app strategy is unknown_app; with it, the fee rule.
	u := newLister(t, n, Config{StartBlock: 1, ProbedTokens: probed, Confirmations: &zero})
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	got := map[DiscardReason]int{}
	for _, r := range []DiscardReason{ReasonApp, ReasonTxOriginGuard, ReasonUnsupportedOpcode, ReasonTokenCount, ReasonTokenUnprobed,
		ReasonTokenExcluded, ReasonMakerMismatch, ReasonNonCanonical, ReasonDockedInWindow} {
		got[r] = int(recorder.get(MetricNameForTest + "|" + string(r)))
	}
	require.Equal(t, want[ReasonTxOriginGuard], got[ReasonTxOriginGuard])
	require.Equal(t, want[ReasonUnsupportedOpcode], got[ReasonUnsupportedOpcode])
	require.Equal(t, want[ReasonTokenCount], got[ReasonTokenCount])
	require.Equal(t, want[ReasonTokenUnprobed], got[ReasonTokenUnprobed])
	require.Equal(t, want[ReasonTokenExcluded], got[ReasonTokenExcluded])
	require.Equal(t, want[ReasonMakerMismatch], got[ReasonMakerMismatch])
	require.Equal(t, want[ReasonNonCanonical], got[ReasonNonCanonical])
	require.Equal(t, want[ReasonDockedInWindow], got[ReasonDockedInWindow])
	// The launch fee strategy: unknown_app here (3 in total with the other app discards).
	require.Equal(t, float64(3), recorder.get(MetricNameForTest+"|"+string(ReasonApp)))
	require.Len(t, pools, 2, "the two admitted strategies")
	// token_unprobed reports both tokens of the strategy it left out (the multicall is not served here,
	// so the fillable stays zero).
	require.Equal(t, float64(1), recorder.get("token_unprobed|0x4444444444444444444444444444444444444444"),
		"the docked strategy with the same unprobed token is not reported")
	require.Equal(t, float64(2), recorder.get("token_unprobed|"+tokenA), "the unprobed and the excluded strategy both hold token A")

	// With the launch router configured the fee strategy is refused for its token.
	recorder.reset()
	u2 := newLister(t, n, Config{StartBlock: 1, ProbedTokens: probed, LaunchRouterAddress: testLaunchHex})
	_, _, err = u2.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, float64(1), recorder.get(MetricNameForTest+"|"+string(ReasonZeroTransferRevertsFees)))
	require.Equal(t, float64(1), recorder.get(MetricNameForTest+"|"+string(ReasonApp)), "only the foreign app: the launch app now passes the app check")
}

// MetricNameForTest is the metric name the recorder keys discards by.
const MetricNameForTest = metrics.MetricNameDiscardedStrategy

func TestListerDockedLaterBecomesTrackerState(t *testing.T) {
	recorder.reset()
	n := newFakeNode(t, 100)
	o := orderOf(testMaker, prog(iFlat(1), iSwap()))
	h := ship(t, n, 10, txHash(1), 0, o, testRouter, tokenA, tokenB)
	u := newLister(t, n, Config{StartBlock: 1, LogWindow: 20})
	pools, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1, "docked in a later window: the pool is created and the tracker archives it")
	n.add(dockedLog(t, 50, 0, txHash(2), testMaker, testRouter, h))
	again, _, err := u.GetNewPools(context.Background(), meta)
	require.NoError(t, err)
	require.Empty(t, again)
}

func TestListerCursorAndConfirmations(t *testing.T) {
	n := newFakeNode(t, 1000)
	o1 := orderOf(testMaker, prog(iFlat(1), iSwap()))
	ship(t, n, 100, txHash(1), 0, o1, testRouter, tokenA, tokenB)
	three := uint64(3)
	u := newLister(t, n, Config{StartBlock: 50, LogWindow: 400, Confirmations: &three})

	pools, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	var m Metadata
	require.NoError(t, json.Unmarshal(meta, &m))
	require.Equal(t, uint64(997), m.LastScannedBlock, "the cursor stays confirmations behind head")
	require.Len(t, n.requests, 3, "windows [50,449] [450,849] [850,997]")
	require.Equal(t, [2]uint64{850, 997}, n.requests[2])

	// Nothing new: same cursor, no request.
	before := len(n.requests)
	pools, meta2, err := u.GetNewPools(context.Background(), meta)
	require.NoError(t, err)
	require.Empty(t, pools)
	require.Equal(t, meta, meta2)
	require.Len(t, n.requests, before)

	// A new strategy past the cursor is the only one listed next.
	n.mu.Lock()
	n.head = 1100
	n.mu.Unlock()
	ship(t, n, 1050, txHash(2), 0, orderOf(testMaker, prog(iFlat(2), iSwap())), testRouter, tokenA, tokenB)
	pools, meta3, err := u.GetNewPools(context.Background(), meta)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.NoError(t, json.Unmarshal(meta3, &m))
	require.Equal(t, uint64(1097), m.LastScannedBlock)

	// Head below the confirmations: nothing to do.
	young := newFakeNode(t, 2)
	pools, metaY, err := newLister(t, young, Config{Confirmations: &three}).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, pools)
	require.Nil(t, metaY)

}

func TestListerWindowsPerRunAndFailures(t *testing.T) {
	n := newFakeNode(t, 1000)
	ship(t, n, 30, txHash(1), 0, orderOf(testMaker, prog(iFlat(1), iSwap())), testRouter, tokenA, tokenB)
	ship(t, n, 330, txHash(2), 0, orderOf(testMaker, prog(iFlat(2), iSwap())), testRouter, tokenA, tokenB)
	ship(t, n, 630, txHash(3), 0, orderOf(testMaker, prog(iFlat(3), iSwap())), testRouter, tokenA, tokenB)
	u := newLister(t, n, Config{StartBlock: 1, LogWindow: 300, MaxWindowsPerRun: 2})

	// Two windows per run: the cursor moves by two windows.
	pools, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	var m Metadata
	require.NoError(t, json.Unmarshal(meta, &m))
	require.Equal(t, uint64(600), m.LastScannedBlock)

	// A failure in the second window of a run keeps what the first found and moves the cursor to its end.
	n.mu.Lock()
	n.failFrom = 901
	n.mu.Unlock()
	pools, meta2, err := u.GetNewPools(context.Background(), meta)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.NoError(t, json.Unmarshal(meta2, &m))
	require.Equal(t, uint64(900), m.LastScannedBlock)

	// A failure in the first window returns the error and the cursor it was given.
	pools, meta3, err := u.GetNewPools(context.Background(), meta2)
	require.Error(t, err)
	require.Empty(t, pools)
	require.Equal(t, meta2, meta3)

	_, _, err = u.GetNewPools(context.Background(), []byte("{"))
	require.Error(t, err)

	down := newFakeNode(t, 10)
	u2 := newLister(t, down, Config{})
	down.Close()
	_, meta4, err := u2.GetNewPools(context.Background(), []byte(`{"lastScannedBlock":3}`))
	require.Error(t, err)
	require.Equal(t, []byte(`{"lastScannedBlock":3}`), meta4)
}

func TestListerBisectsRangeErrors(t *testing.T) {
	recorder.reset()
	n := newFakeNode(t, 200) // block 37 is more than a log window below the head
	n.maxSpan = 8
	n.badBlock[37] = true
	o := orderOf(testMaker, prog(iFlat(1), iSwap()))
	ship(t, n, 5, txHash(1), 0, o, testRouter, tokenA, tokenB)
	ship(t, n, 60, txHash(2), 0, orderOf(testMaker, prog(iFlat(2), iSwap())), testRouter, tokenA, tokenB)
	u := newLister(t, n, Config{StartBlock: 1, LogWindow: 64})
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 2, "both strategies found around the block the node cannot serve")
	require.Equal(t, float64(1), recorder.get("skipped_unservable_blocks"), "the unservable block is counted")
	// Bisection isolates the unservable block: each request that holds it is smaller than the one before,
	// the single-block request is the last of them and nothing spans block 37 after it.
	alone := -1
	prevSpan := uint64(1 << 62)
	for i, r := range n.requests {
		if r[0] > 37 || r[1] < 37 {
			continue
		}
		require.Negative(t, alone, "no request holds block 37 after the single-block request")
		span := r[1] - r[0] + 1
		require.Less(t, span, prevSpan)
		prevSpan = span
		if span == 1 {
			alone = i
		}
	}
	require.GreaterOrEqual(t, alone, 0, "block 37 was requested alone")

	// An error that is not about the range is not bisected.
	bad := newFakeNode(t, 64)
	bad.failFrom = 1
	_, _, err = newLister(t, bad, Config{StartBlock: 1}).GetNewPools(context.Background(), nil)
	require.ErrorContains(t, err, "exploded")
	require.Len(t, bad.requests, 1)

	// A cancelled context stops the bisection.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	limited := newFakeNode(t, 64)
	limited.maxSpan = 1
	_, _, err = newLister(t, limited, Config{StartBlock: 1, LogWindow: 64}).GetNewPools(ctx, nil)
	require.Error(t, err)
}

func TestListerIgnoresBadLogs(t *testing.T) {
	n := newFakeNode(t, 100)
	o := orderOf(testMaker, prog(iFlat(1), iSwap()))
	ship(t, n, 10, txHash(1), 0, o, testRouter, tokenA, tokenB)
	removed := shippedLog(t, 11, 0, txHash(2), testMaker, testRouter, o.Hash(), o.Encode())
	removed.Removed = true
	noTopic := types.Log{Address: aquaAddr, Topics: []common.Hash{}, BlockNumber: 12}
	garbageShipped := mkLog("Shipped", 13, 0, txHash(3), []byte{1, 2, 3})
	garbagePushed := mkLog("Pushed", 13, 1, txHash(3), []byte{1})
	garbageDocked := mkLog("Docked", 13, 2, txHash(3), []byte{1})
	unknownTopic := types.Log{Address: aquaAddr, Topics: []common.Hash{common.HexToHash("0x01")}, BlockNumber: 14}
	n.add(removed, noTopic, garbageShipped, garbagePushed, garbageDocked, unknownTopic)
	pools, _, err := newLister(t, n, Config{StartBlock: 1}).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
}

func TestListerDeduplicatesPools(t *testing.T) {
	n := newFakeNode(t, 100)
	o := orderOf(testMaker, prog(iFlat(1), iSwap()))
	ship(t, n, 10, txHash(1), 0, o, testRouter, tokenA, tokenB)
	ship(t, n, 12, txHash(2), 0, o, testRouter, tokenA, tokenB) // the same strategy announced twice
	pools, _, err := newLister(t, n, Config{StartBlock: 1}).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
}

func TestReportUnprobedFillable(t *testing.T) {
	recorder.reset()
	chain := newFakeChain()
	maker := testMaker
	o := orderOf(maker, prog(iFlat(1), iSwap()))
	hash := o.Hash()
	chain.raw[rawKey(maker, testRouter, hash, common.HexToAddress(tokenA))] = [2]*big.Int{big.NewInt(500), big.NewInt(2)}
	chain.raw[rawKey(maker, testRouter, hash, common.HexToAddress(tokenB))] = [2]*big.Int{big.NewInt(900), big.NewInt(2)}
	chain.setWallet(maker.Hex(), tokenA, uint256.NewInt(400), uint256.NewInt(1000))
	chain.setWallet(maker.Hex(), tokenB, uint256.NewInt(2000), uint256.NewInt(300))
	// A docked strategy adds nothing.
	o2 := orderOf(maker, prog(iFlat(2), iSwap()))
	chain.raw[rawKey(maker, testRouter, o2.Hash(), common.HexToAddress(tokenA))] = [2]*big.Int{big.NewInt(500), big.NewInt(tokensCountDocked)}
	chain.raw[rawKey(maker, testRouter, o2.Hash(), common.HexToAddress(tokenB))] = [2]*big.Int{big.NewInt(500), big.NewInt(tokensCountDocked)}

	tr, stub := chain.tracker(t)
	_ = tr
	u := &PoolsListUpdater{cfg: &settings{dexId: DexType, aqua: aquaAddr, maxUnprobed: DefaultMaxUnprobedReports}, ethrpcClient: stub.Client()}
	pair := [2]common.Address{common.HexToAddress(tokenA), common.HexToAddress(tokenB)}
	u.reportUnprobed(context.Background(), []unprobedStrategy{
		{maker: maker, app: testRouter, hash: hash, tokens: pair},
		{maker: maker, app: testRouter, hash: o2.Hash(), tokens: pair},
	})
	require.Equal(t, float64(2), recorder.get("token_unprobed|"+tokenA))
	require.Equal(t, float64(400), recorder.get("token_unprobed_fillable|"+tokenA), "min(raw 500, balance 400, allowance 1000)")
	require.Equal(t, float64(300), recorder.get("token_unprobed_fillable|"+tokenB), "min(raw 900, balance 2000, allowance 300)")
	u.reportUnprobed(context.Background(), nil)
}

// A throttled, timed-out or dropped request never skips a block and never moves the cursor: the error
// comes back with the cursor it was given, and no skipped block is counted.
func TestListerTransientErrorsKeepCursor(t *testing.T) {
	cursor := []byte(`{"lastScannedBlock":3}`)
	faults := map[string]fault{
		"http 429":        {status: http.StatusTooManyRequests},
		"429 message":     {message: "429 Too Many Requests: slow down"},
		"rate limit":      {message: "rate limit exceeded"},
		"compute units":   {message: "Your app has exceeded its compute units per second capacity"},
		"client timeout":  {message: `Post "http://node": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`},
		"transport drop":  {drop: true},
		"connection loss": {message: "read tcp 10.0.0.1:5000->10.0.0.2:8545: connection reset by peer"},
		"generic":         {message: "upstream exploded"},
	}
	for name, f := range faults {
		t.Run(name, func(t *testing.T) {
			recorder.reset()
			// The first request of the window fails: the window is not split.
			whole := newFakeNode(t, 64)
			whole.failFrom, whole.fault = 1, f
			ship(t, whole, 5, txHash(1), 0, orderOf(testMaker, prog(iFlat(1), iSwap())), testRouter, tokenA, tokenB)
			pools, meta, err := newLister(t, whole, Config{StartBlock: 1, LogWindow: 64}).GetNewPools(context.Background(), cursor)
			require.Error(t, err)
			require.Empty(t, pools)
			require.Equal(t, cursor, meta, "the cursor stays")
			for _, r := range whole.requestLog() {
				require.Equal(t, [2]uint64{4, 64}, r, "not a range error: the window is never split")
			}

			// The failure comes after the window was split down to a single block: still no skip.
			deep := newFakeNode(t, 64)
			deep.maxSpan, deep.failBlock, deep.fault = 8, 37, f
			ship(t, deep, 60, txHash(2), 0, orderOf(testMaker, prog(iFlat(1), iSwap())), testRouter, tokenA, tokenB)
			pools, meta, err = newLister(t, deep, Config{StartBlock: 1, LogWindow: 64}).GetNewPools(context.Background(), cursor)
			require.Error(t, err)
			require.Empty(t, pools)
			require.Equal(t, cursor, meta)
			require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))
		})
	}

	// A single-block window that fails is an error too, whatever the message says about ranges.
	recorder.reset()
	single := newFakeNode(t, 40)
	single.failBlock, single.fault = 37, fault{message: "429 Too Many Requests: block range limit"}
	_, meta, err := newLister(t, single, Config{StartBlock: 37, LogWindow: 1}).GetNewPools(context.Background(), []byte(`{"lastScannedBlock":36}`))
	require.Error(t, err)
	require.Equal(t, []byte(`{"lastScannedBlock":36}`), meta)
	require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))

	// A refused range that is a limit message still splits, and the unservable block is the only skip.
	recorder.reset()
	n := newFakeNode(t, 200)
	n.maxSpan = 8
	n.badBlock[37] = true
	pools, _, err := newLister(t, n, Config{StartBlock: 1, LogWindow: 64}).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, pools)
	require.Equal(t, float64(1), recorder.get("skipped_unservable_blocks"))
}

// Real provider messages. A range error splits the window; a transient error returns with the cursor kept.
// Server-side timeouts that ask for a smaller range are range errors, a client that gave up is transient.
func TestRangeErrorClassification(t *testing.T) {
	t.Parallel()
	const (
		rng       = "range"
		transient = "transient"
		neither   = "neither" // not a range refusal and not marked transient: returned all the same
	)
	for _, tc := range []struct{ kind, msg string }{
		{rng, "Log response size exceeded. You can make eth_getLogs requests with up to a 2K block range and no limit on the response size, or you can request any block range with a cap of 10K logs in the response. Based on your parameters, this block range should work: [0x1, 0x7d0]"},
		{rng, "Under the Free tier plan, you can make eth_getLogs requests with up to a 10 block range. Based on your parameters, this block range should work: [0x1, 0xa]. Upgrade to PAYG for expanded block range."},
		{rng, "you can make eth_getLogs requests with up to a 10 block range"},
		{rng, "exceed maximum block range: 50000"},
		{rng, "block range too large"},
		{rng, "block range is too wide"},
		{rng, "eth_getLogs is limited to a 10,000 range"},
		{rng, "query exceeds max block range 100000"},
		{rng, "query returned more than 10000 results"},
		{rng, "query returned more than 10000 results. Try with this block range [0x1, 0x2]."},
		{rng, "Block range limit exceeded"},
		{rng, "ranges over 10000 blocks are not supported on freetier"},
		{rng, "requested too many blocks from 1 to 100000, maximum is set to 2048"},
		{rng, "block range greater than 10000 max"},
		{rng, "eth_getLogs and eth_newFilter are limited to a 10,000 blocks range"},
		{rng, "max results exceeded"},
		{rng, "response size should not greater than 10000000 bytes"},
		{rng, "Request exceeds defined limit"},
		{rng, "maximum block range is 1000"},
		{rng, "exceeded max allowed range"},
		{rng, "invalid block range"},
		{rng, "query exceeds max results 20000, retry with the range 1-100"},
		{rng, "getLogs: block range limit (1000) exceeded"},
		{rng, "413 Request Entity Too Large"},
		{rng, "limit exceeded"},
		{rng, "query exceeds limit of results"},
		{rng, "Log response size exceeded"},
		// Server-side timeouts: the window is too heavy for the node, so it splits.
		{rng, "Query timeout exceeded. Consider reducing your block range."},
		{rng, "504 Gateway Timeout"},
		{rng, "504 Gateway Timeout: <html>upstream request timeout</html>"},
		// A block number in hex that spells a status code is not that status code.
		{rng, "Log response size exceeded. Based on your parameters, this block range should work: [0x1, 0x429]"},
		// Throttling, client-side timeouts and transport failures.
		{transient, "429 Too Many Requests: {\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32005,\"message\":\"limit exceeded\"}}"},
		{transient, "429 Too Many Requests"},
		{transient, "project ID request rate exceeded"},
		{transient, "Too many requests, reason: call rate limit exhausted, retry in 10m0s"},
		{transient, "daily request count exceeded, request rate limited"},
		{transient, "capacity exceeded"},
		{transient, "Your app has exceeded its compute units per second capacity"},
		{transient, "exceeded the quota usage"},
		{transient, "request timed out"},
		{transient, "rate limit exceeded"},
		{transient, "daily request limit reached"},
		{transient, "logs request limit exceeded"}, // reads as a quota; fail safe, keep the cursor
		{transient, `Post "http://node": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`},
		{transient, `Client.Timeout exceeded while awaiting headers`},
		{transient, "context deadline exceeded"},
		{transient, `Post "http://node": dial tcp 10.0.0.1:8545: i/o timeout`},
		{transient, "net/http: TLS handshake timeout"},
		{transient, "dial tcp: connection refused"},
		{transient, "read tcp 10.0.0.1:5000->10.0.0.2:8545: connection reset by peer"},
		{transient, "Post \"http://node\": EOF"},
		{neither, "503 Service Unavailable"},
		{neither, "upstream connect error or disconnect/reset before headers"},
		{neither, "upstream exploded"},
	} {
		err := errors.New(tc.msg)
		require.Equal(t, tc.kind == rng, isRangeError(err), tc.msg)
		require.Equal(t, tc.kind == transient, isTransientError(err), tc.msg)
	}
	require.True(t, isUnservableBlock(errors.New("invalid block range")))
	require.True(t, isUnservableBlock(errors.New("invalid block range params")), "Polygon bor")
	require.False(t, isUnservableBlock(errors.New("429 Too Many Requests: invalid block range")))
	require.False(t, isUnservableBlock(errors.New("upstream exploded")))
	require.True(t, isTransientError(timeoutErr{}))
	require.False(t, isRangeError(timeoutErr{}))
	require.True(t, isTransientError(rpc.HTTPError{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests"}))
	gateway := rpc.HTTPError{StatusCode: http.StatusGatewayTimeout, Status: "504 Gateway Timeout"}
	require.True(t, isRangeError(gateway))
	require.False(t, isTransientError(gateway))
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// withState writes a refreshed state into a listed pool, the way the tracker would.
func withState(t *testing.T, p entity.Pool, raw0, raw1 *uint256.Int) entity.Pool {
	t.Helper()
	extra, err := json.Marshal(Extra{RawBalance0: raw0, RawBalance1: raw1, RawBlock: 100, WalletBalance0: e18(1_000_000),
		WalletBalance1: e18(1_000_000), Allowance0: unlimited, Allowance1: unlimited, WalletBlock: 100})
	require.NoError(t, err)
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{raw0.Dec(), raw1.Dec()}
	return p
}

// The launch flag decides whether the simulator refuses a protocol fee the maker cannot cover, the one
// place where a wrong flag quotes a swap the launch router reverts.
func TestListerSetsLaunchFlag(t *testing.T) {
	recorder.reset()
	n := newFakeNode(t, 100)
	feeOrder := orderOf(testMaker, prog(iSalt(1), iProtocol(100_000_000, testRecipient), iSwap()))
	launchHash := ship(t, n, 10, txHash(1), 0, feeOrder, testLaunch, tokenA, tokenB)
	routerHash := ship(t, n, 11, txHash(2), 0, feeOrder, testRouter, tokenA, tokenB)
	require.Equal(t, launchHash, routerHash, "one order shipped on both routers")

	list := func(launchAddr string) (map[string]entity.Pool, error) {
		u := newLister(t, n, Config{StartBlock: 1, LaunchRouterAddress: launchAddr})
		pools, _, err := u.GetNewPools(context.Background(), nil)
		byId := map[string]entity.Pool{}
		for _, p := range pools {
			byId[p.Address] = p
		}
		return byId, err
	}

	pools, err := list(testLaunchHex)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	launchPool, routerPool := pools[poolId(testLaunch, testMaker, launchHash)], pools[poolId(testRouter, testMaker, routerHash)]
	require.NotEmpty(t, launchPool.Address)
	require.NotEmpty(t, routerPool.Address)
	require.True(t, decodeStatic(t, launchPool).Launch, "launch-app strategy")
	require.False(t, decodeStatic(t, routerPool).Launch, "current-router strategy")
	require.Equal(t, hexutil.Encode(testLaunch[:]), decodeStatic(t, launchPool).App)

	// The maker's strategy balance on the in side cannot cover the 10% fee of 1,000,000: the launch router
	// reverts, the current router skips the pull.
	raw := [2]*uint256.Int{uint256.NewInt(99_999), e18(10)}
	amount := uint256.NewInt(1_000_000)
	launchSim, err := NewPoolSimulator(withState(t, launchPool, raw[0], raw[1]))
	require.NoError(t, err)
	_, err = quoteOut(launchSim, 0, amount, nil)
	require.ErrorIs(t, err, ErrFeeShortfall)
	routerSim, err := NewPoolSimulator(withState(t, routerPool, raw[0], raw[1]))
	require.NoError(t, err)
	res, err := quoteOut(routerSim, 0, amount, nil)
	require.NoError(t, err)
	require.Equal(t, "0", res.Fee.Amount.String(), "the current router skips the pull")
	// With enough on the in side both quote, and the fee is collected.
	covered := [2]*uint256.Int{uint256.NewInt(100_000), e18(10)}
	launchSim, err = NewPoolSimulator(withState(t, launchPool, covered[0], covered[1]))
	require.NoError(t, err)
	res, err = quoteOut(launchSim, 0, amount, nil)
	require.NoError(t, err)
	require.Equal(t, "100000", res.Fee.Amount.String())

	// No launch router configured: the launch-app strategy is refused as an unknown app, the other stays.
	recorder.reset()
	pools, err = list("")
	require.NoError(t, err)
	require.Len(t, pools, 1)
	for _, p := range pools {
		require.False(t, decodeStatic(t, p).Launch)
		require.Equal(t, hexutil.Encode(testRouter[:]), decodeStatic(t, p).App)
	}
	require.Equal(t, float64(1), recorder.get(MetricNameForTest+"|"+string(ReasonApp)))
}

// The lister writes the tokens in ascending address order whatever order the Pushed logs come in; the
// pool id does not depend on it, the tracker reads the raw and wallet balances for that order and the
// simulator quotes the same both ways.
func TestListerCanonicalTokenOrder(t *testing.T) {
	recorder.reset()
	order := orderOf(testMaker, prog(iConcentrate(sqrtPrice(e18(1)), sqrtPrice(e18(4))), iFlat(10_000_000), iSwap(), iSalt(1)))
	listed := func(first, second string) entity.Pool {
		n := newFakeNode(t, 100)
		ship(t, n, 10, txHash(1), 0, order, testRouter, first, second)
		pools, _, err := newLister(t, n, Config{StartBlock: 1}).GetNewPools(context.Background(), nil)
		require.NoError(t, err)
		require.Len(t, pools, 1)
		return pools[0]
	}
	asc, desc := listed(tokenA, tokenB), listed(tokenB, tokenA)
	require.Equal(t, asc.Address, desc.Address, "the pool id does not depend on the token order")
	require.Equal(t, asc.StaticExtra, desc.StaticExtra)
	for _, p := range []entity.Pool{asc, desc} {
		require.Equal(t, []string{tokenA, tokenB}, []string{p.Tokens[0].Address, p.Tokens[1].Address})
	}

	// The tracker reads the balances of the tokens in the pool's order.
	chain := newFakeChain()
	static := decodeStatic(t, desc)
	hash := common.HexToHash(static.StrategyHash)
	chain.raw[rawKey(testMaker, testRouter, hash, common.HexToAddress(tokenA))] = [2]*big.Int{e18(1000).ToBig(), big.NewInt(2)}
	chain.raw[rawKey(testMaker, testRouter, hash, common.HexToAddress(tokenB))] = [2]*big.Int{e18(3000).ToBig(), big.NewInt(2)}
	chain.setWallet(testMaker.Hex(), tokenA, e18(500), unlimited)
	chain.setWallet(testMaker.Hex(), tokenB, e18(7000), unlimited)
	tr, _ := chain.tracker(t)
	var tracked [2]entity.Pool
	for i, p := range []entity.Pool{asc, desc} {
		var err error
		tracked[i], err = tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
	}
	require.Equal(t, tracked[0].Extra, tracked[1].Extra)
	require.Equal(t, entity.PoolReserves{e18(500).Dec(), e18(3000).Dec()}, tracked[1].Reserves, "min(raw, wallet, allowance) per token, in token order")
	x := decodeExtra(t, tracked[1])
	require.Equal(t, e18(1000).Dec(), x.RawBalance0.Dec())
	require.Equal(t, e18(3000).Dec(), x.RawBalance1.Dec())
	require.Equal(t, e18(500).Dec(), x.WalletBalance0.Dec())
	require.Equal(t, e18(7000).Dec(), x.WalletBalance1.Dec())

	// The quote of each direction is the same from both listings, and the directions differ.
	simAsc, err := NewPoolSimulator(tracked[0])
	require.NoError(t, err)
	simDesc, err := NewPoolSimulator(tracked[1])
	require.NoError(t, err)
	var outs []string
	for in := range 2 {
		a, err := quoteOut(simAsc, in, e18(10), nil)
		require.NoError(t, err)
		d, err := quoteOut(simDesc, in, e18(10), nil)
		require.NoError(t, err)
		require.Equal(t, a.TokenAmountOut.Token, d.TokenAmountOut.Token)
		require.Equal(t, a.TokenAmountOut.Amount.String(), d.TokenAmountOut.Amount.String())
		require.Equal(t, a.Fee.Amount.String(), d.Fee.Amount.String())
		outs = append(outs, a.TokenAmountOut.Amount.String())
	}
	require.NotEqual(t, outs[0], outs[1], "the two directions are not symmetric on this state")
	require.Equal(t, tokenB, simDesc.Info.Tokens[1])
}

// The report reads the fillable of at most maxUnprobed strategies, in multicalls of unprobedBatch; the
// strategies past the bound are still counted.
func TestReportUnprobedIsBoundedAndBatched(t *testing.T) {
	recorder.reset()
	chain := newFakeChain()
	const total, bound = 120, 110
	pair := [2]common.Address{common.HexToAddress(tokenA), common.HexToAddress(tokenB)}
	chain.setWallet(testMaker.Hex(), tokenA, uint256.NewInt(1000), uint256.NewInt(1000))
	chain.setWallet(testMaker.Hex(), tokenB, uint256.NewInt(1000), uint256.NewInt(1000))
	var items []unprobedStrategy
	for i := range total {
		hash := common.BigToHash(big.NewInt(int64(i + 1)))
		for _, tk := range pair {
			chain.raw[rawKey(testMaker, testRouter, hash, tk)] = [2]*big.Int{big.NewInt(10), big.NewInt(2)}
		}
		items = append(items, unprobedAt(hash, pair))
	}
	_, stub := chain.tracker(t)
	u := &PoolsListUpdater{cfg: &settings{dexId: DexType, aqua: aquaAddr, maxUnprobed: bound}, ethrpcClient: stub.Client()}
	u.reportUnprobed(context.Background(), items)

	calls := stub.Targets()
	require.Len(t, calls, 3, "110 strategies in batches of 50, 50, 10")
	require.Len(t, calls[0], 6*unprobedBatch)
	require.Len(t, calls[2], 6*10)
	require.Equal(t, float64(total), recorder.get("token_unprobed|"+tokenA), "every strategy is counted")
	require.Equal(t, float64(bound*10), recorder.get("token_unprobed_fillable|"+tokenA), "the fillable of the bounded strategies only")
}

func unprobedAt(hash common.Hash, pair [2]common.Address) unprobedStrategy {
	return unprobedStrategy{maker: testMaker, app: testRouter, hash: hash, tokens: pair}
}

// Scan skips only blocks the node cannot serve because it pruned them. HyperEVM nodes send the same
// "invalid block range" for a block above their head, so an unservable block close to the confirmed head
// is an error: the cursor stays and no block is counted as skipped.
func TestListerKeepsCursorForUnservableBlockNearHead(t *testing.T) {
	zero, two := uint64(0), uint64(2)
	o := orderOf(testMaker, prog(iFlat(1), iSwap()))
	for name, tc := range map[string]struct {
		head, bad, window uint64
		confirmations     *uint64
	}{
		"head block, confirmations 0":     {head: 100, bad: 100, window: 1000, confirmations: &zero},
		"head block, small window":        {head: 100, bad: 100, window: 10, confirmations: &zero},
		"confirmed head, confirmations 2": {head: 102, bad: 100, window: 1000, confirmations: &two},
		"within one window of the head":   {head: 100, bad: 91, window: 10, confirmations: &zero},
	} {
		t.Run(name, func(t *testing.T) {
			recorder.reset()
			n := newFakeNode(t, tc.head)
			ship(t, n, tc.bad, txHash(1), 0, o, testRouter, tokenA, tokenB)
			n.badBlock[tc.bad] = true // the serving backend has not got this block yet
			cursor := []byte(`{"lastScannedBlock":90}`)
			u := newLister(t, n, Config{StartBlock: 1, LogWindow: tc.window, Confirmations: tc.confirmations})
			pools, meta, err := u.GetNewPools(context.Background(), cursor)
			require.ErrorContains(t, err, "invalid block range")
			require.Empty(t, pools)
			require.Equal(t, cursor, meta, "the cursor stays")
			require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))

			// The backend catches up: the next run finds the strategy it would have lost.
			delete(n.badBlock, tc.bad)
			pools, meta, err = u.GetNewPools(context.Background(), meta)
			require.NoError(t, err)
			require.Len(t, pools, 1)
			var m Metadata
			require.NoError(t, json.Unmarshal(meta, &m))
			require.Equal(t, uint64(100), m.LastScannedBlock)
			require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))
		})
	}

	// A later window that fails keeps what the finished windows found, and the cursor stops at their end.
	recorder.reset()
	n := newFakeNode(t, 100)
	ship(t, n, 75, txHash(1), 0, o, testRouter, tokenA, tokenB)
	n.badBlock[95] = true
	pools, meta, err := newLister(t, n, Config{StartBlock: 1, LogWindow: 10, Confirmations: &zero}).
		GetNewPools(context.Background(), []byte(`{"lastScannedBlock":70}`))
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, []byte(`{"lastScannedBlock":90}`), meta)
	require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))
}

// A block a log window or more below the head is pruned history, not a block the node lacks: it is skipped
// and counted, and the scan goes on to the head.
func TestListerSkipsUnservableBlockFarBelowHead(t *testing.T) {
	zero := uint64(0)
	o := orderOf(testMaker, prog(iFlat(1), iSwap()))
	for name, bad := range map[string]uint64{
		"far below":                      40,
		"exactly one window below":       90, // bad + window == target
		"one block short of that margin": 91, // bad + window == target + 1: an error, the cursor stops before it
	} {
		t.Run(name, func(t *testing.T) {
			recorder.reset()
			n := newFakeNode(t, 100)
			ship(t, n, 99, txHash(1), 0, o, testRouter, tokenA, tokenB)
			n.badBlock[bad] = true
			u := newLister(t, n, Config{StartBlock: 1, LogWindow: 10, MaxWindowsPerRun: 20, Confirmations: &zero})
			cursor := []byte(`{"lastScannedBlock":30}`)
			pools, meta, err := u.GetNewPools(context.Background(), cursor)
			require.NoError(t, err)
			if bad == 91 {
				require.Empty(t, pools)
				require.Equal(t, []byte(`{"lastScannedBlock":90}`), meta)
				require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))
				return
			}
			require.Len(t, pools, 1, "the strategy at the head is still found")
			require.Equal(t, []byte(`{"lastScannedBlock":100}`), meta)
			require.Equal(t, float64(1), recorder.get("skipped_unservable_blocks"))
		})
	}
}

// A range refusal made of a server timeout splits the window, like any other range error. Single blocks no
// longer skip, so the split ends in the logs or in an error.
func TestListerBisectsServerTimeouts(t *testing.T) {
	zero := uint64(0)
	for name, f := range map[string]fault{
		"alchemy query timeout": {message: "Query timeout exceeded. Consider reducing your block range."},
		"gateway timeout body":  {message: "504 Gateway Timeout"},
		"gateway timeout http":  {status: http.StatusGatewayTimeout},
	} {
		t.Run(name, func(t *testing.T) {
			recorder.reset()
			n := newFakeNode(t, 64)
			n.maxSpan, n.spanFault = 8, &f
			ship(t, n, 5, txHash(1), 0, orderOf(testMaker, prog(iFlat(1), iSwap())), testRouter, tokenA, tokenB)
			ship(t, n, 60, txHash(2), 0, orderOf(testMaker, prog(iFlat(2), iSwap())), testRouter, tokenA, tokenB)
			pools, meta, err := newLister(t, n, Config{StartBlock: 1, LogWindow: 64, Confirmations: &zero}).
				GetNewPools(context.Background(), nil)
			require.NoError(t, err)
			require.Len(t, pools, 2)
			require.Equal(t, []byte(`{"lastScannedBlock":64}`), meta)
			require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))
			for _, r := range n.requestLog() {
				if r[1]-r[0]+1 <= 8 {
					continue
				}
				require.Less(t, r[1]-r[0]+1, uint64(65), "no request is wider than the window")
			}
			require.Greater(t, len(n.requestLog()), 1)
		})
	}

	// A window that times out down to one block is an error, and the cursor stays.
	recorder.reset()
	n := newFakeNode(t, 64)
	n.failBlock, n.fault = 20, fault{message: "Query timeout exceeded. Consider reducing your block range."}
	cursor := []byte(`{"lastScannedBlock":3}`)
	_, meta, err := newLister(t, n, Config{StartBlock: 1, LogWindow: 64, Confirmations: &zero}).GetNewPools(context.Background(), cursor)
	require.Error(t, err)
	require.Equal(t, cursor, meta)
	require.Equal(t, float64(0), recorder.get("skipped_unservable_blocks"))
}

type excludedOrder struct {
	ChainId       int      `json:"chainId"`
	App           string   `json:"app"`
	Maker         string   `json:"maker"`
	StrategyHash  string   `json:"strategyHash"`
	Tokens        []string `json:"tokens"`
	Order         string   `json:"order"`
	ProgramOffset int      `json:"programOffset"`
}

// A real HyperEVM order (a swap on the current router) with no tx.origin guard: it is excluded by its
// post-transfer-out hook and, in the program behind the hook slice, by the extruction instruction.
func TestHyperEVMHookExtructionOrderExcluded(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("testdata/hyperevm_hook_order.json")
	require.NoError(t, err)
	var v excludedOrder
	require.NoError(t, json.Unmarshal(b, &v))

	raw := hexutil.MustDecode(v.Order)
	order, err := DecodeOrder(raw)
	require.NoError(t, err)
	require.Equal(t, v.StrategyHash, order.Hash().Hex(), "the canonical encoding hashes to the strategy hash")
	require.Equal(t, common.HexToAddress(v.Maker), order.Maker)
	require.True(t, traitsBit(&order.Traits, traitsUseAqua))
	require.True(t, traitsBit(&order.Traits, 249), "post-transfer-out hook")
	require.True(t, traitsBit(&order.Traits, 245), "with a target")
	require.Equal(t, uint64(v.ProgramOffset), sliceIndex(&order.Traits, 3))

	// Rule 3: the hook flag refuses the order first.
	_, err = ParseOrder(order, false)
	requireReason(t, err, ReasonHooks, "hook flag")

	// Rule 4: the program behind the hook slice holds extruction (0x20) and no 0x21.
	program := order.Data[v.ProgramOffset:]
	require.Equal(t, byte(opXYCSwap), program[0])
	_, err = parseProgram(program, false)
	requireReason(t, err, ReasonUnsupportedOpcode, "extruction")
	_, err = parseProgram(program, true)
	requireReason(t, err, ReasonUnsupportedOpcode, "extruction on the launch router")

	// Through the lister: with both tokens probed the order still never becomes a pool.
	u := &PoolsListUpdater{cfg: &settings{dexId: DexType, router: common.HexToAddress(v.App),
		probed: map[string]string{v.Tokens[0]: ProbeOK, v.Tokens[1]: ProbeOK}}}
	s := &shipped{maker: order.Maker, app: common.HexToAddress(v.App), hash: order.Hash(), strategy: raw,
		tokens: []common.Address{common.HexToAddress(v.Tokens[0]), common.HexToAddress(v.Tokens[1])}}
	pool, _, reason := u.admit(s, nil)
	require.Nil(t, pool)
	require.Equal(t, ReasonHooks, reason)
}

// Live Ethereum strategies that carry the tx.origin guard never become pools, and the
// ungated live strategies of the golden vectors all do, none of them holding 0x21.
func TestListerOverLiveStrategies(t *testing.T) {
	recorder.reset()
	var gated []struct {
		Maker        string `json:"maker"`
		App          string `json:"app"`
		StrategyHash string `json:"strategyHash"`
		Strategy     string `json:"strategy"`
		Ops          []int  `json:"ops"`
	}
	b, err := os.ReadFile("testdata/gated_strategies.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &gated))
	require.NotEmpty(t, gated)

	n := newFakeNode(t, 1000)
	for i, g := range gated {
		raw := hexutil.MustDecode(g.Strategy)
		o, err := DecodeOrder(raw)
		require.NoError(t, err)
		require.Equal(t, g.StrategyHash, o.Hash().Hex())
		require.Contains(t, g.Ops, opTxOriginGuard)
		_, err = ParseOrder(o, false)
		requireReason(t, err, ReasonTxOriginGuard, g.StrategyHash)
		tx := txHash(500 + i)
		n.add(shippedLog(t, 10, 0, tx, common.HexToAddress(g.Maker), common.HexToAddress(g.App), common.HexToHash(g.StrategyHash), raw),
			pushedLog(t, 10, 1, tx, common.HexToAddress(g.Maker), common.HexToAddress(g.App), common.HexToHash(g.StrategyHash), tokenA),
			pushedLog(t, 10, 2, tx, common.HexToAddress(g.Maker), common.HexToAddress(g.App), common.HexToHash(g.StrategyHash), tokenB))
	}
	pools, _, err := newLister(t, n, Config{StartBlock: 1}).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, pools, "no strategy with opcode 0x21 becomes a pool")
	require.Equal(t, float64(len(gated)), recorder.get(MetricNameForTest+"|"+string(ReasonTxOriginGuard)))

	// The ungated live strategies.
	ungated := map[string]goldenVector{}
	for _, v := range loadGolden(t) {
		ungated[v.Strategy] = v
	}
	n2 := newFakeNode(t, 1000)
	probed := map[string]string{}
	i := 0
	for strategy, v := range ungated {
		i++
		raw := hexutil.MustDecode(strategy)
		o, err := DecodeOrder(raw)
		require.NoError(t, err)
		for _, tk := range v.Tokens {
			probed[strings.ToLower(tk)] = ProbeOK
		}
		ship(t, n2, 10, txHash(700+i), 0, o, common.HexToAddress(v.App), strings.ToLower(v.Tokens[0]), strings.ToLower(v.Tokens[1]))
	}
	pools, _, err = newLister(t, n2, Config{StartBlock: 1, ProbedTokens: probed, LaunchRouterAddress: testLaunchHex}).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, len(ungated))
	for _, p := range pools {
		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		for _, ins := range sim.program {
			require.NotEqual(t, uint8(opTxOriginGuard), ins.Op)
		}
	}
}

// The live router quotes of the golden vectors are reproduced from the pools the lister lists, with the
// tokens in canonical order whatever order the Pushed logs of the strategy came in: the quote of each
// vector, in its own direction, does not depend on the token order of the listing.
func TestGoldenQuotesThroughLister(t *testing.T) {
	recorder.reset()
	vectors := loadGolden(t)
	n := newFakeNode(t, 1000)
	probed := map[string]string{}
	shippedStrategy := map[string]bool{}
	var descending int
	for i, v := range vectors {
		for _, tk := range v.Tokens {
			probed[strings.ToLower(tk)] = ProbeOK
		}
		if shippedStrategy[v.Strategy] {
			continue
		}
		shippedStrategy[v.Strategy] = true
		o, err := DecodeOrder(hexutil.MustDecode(v.Strategy))
		require.NoError(t, err)
		if common.HexToAddress(v.Tokens[0]).Cmp(common.HexToAddress(v.Tokens[1])) > 0 {
			descending++
		}
		ship(t, n, 10, txHash(900+i), 0, o, common.HexToAddress(v.App), strings.ToLower(v.Tokens[0]), strings.ToLower(v.Tokens[1]))
	}
	require.Positive(t, descending, "the vectors include strategies pushed in descending token order")
	pools, _, err := newLister(t, n, Config{StartBlock: 1, ProbedTokens: probed, LaunchRouterAddress: testLaunchHex}).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, len(shippedStrategy))
	byId := map[string]entity.Pool{}
	for _, p := range pools {
		require.True(t, p.Tokens[0].Address < p.Tokens[1].Address, "canonical ascending order: %s", p.Address)
		byId[p.Address] = p
	}

	compared := 0
	for _, v := range vectors {
		o, err := DecodeOrder(hexutil.MustDecode(v.Strategy))
		require.NoError(t, err)
		p, ok := byId[poolId(common.HexToAddress(v.App), o.Maker, o.Hash())]
		require.True(t, ok)
		// The raw balance of each token, whichever position it has in the listing.
		rawOf := map[string]*uint256.Int{strings.ToLower(v.Tokens[0]): u(v.Raw[0]), strings.ToLower(v.Tokens[1]): u(v.Raw[1])}
		sim, err := NewPoolSimulator(withState(t, p, rawOf[p.Tokens[0].Address], rawOf[p.Tokens[1].Address]))
		require.NoError(t, err)
		tokenIn, tokenOut := strings.ToLower(v.Tokens[v.In]), strings.ToLower(v.Tokens[1-v.In])
		res, err := quoteOut(sim, sim.GetTokenIndex(tokenIn), u(v.AmountIn), nil)
		if err != nil { // the router quotes without the cap of the strategy balance, as in the golden simulator test
			require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
			continue
		}
		compared++
		require.Equal(t, v.AmountOut, res.TokenAmountOut.Amount.String(), "chain %d amountIn %s", v.ChainId, v.AmountIn)
		require.Equal(t, tokenOut, res.TokenAmountOut.Token)
	}
	require.Greater(t, compared, len(vectors)*3/4)
}
