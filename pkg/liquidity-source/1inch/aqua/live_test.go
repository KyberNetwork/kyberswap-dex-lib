package oneinchaqua

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// TestLiveTracker runs the real lister, the real tracker and the simulator against a chain.
//
// Env: ONEINCH_AQUA_RPC_URL (node with eth_getLogs ranges of at least ONEINCH_AQUA_LIVE_WINDOW blocks,
// default 50000), ONEINCH_AQUA_LIVE_START (first block to scan) or ONEINCH_AQUA_LIVE_BLOCKS (how far back
// from head, default 150000), optional ONEINCH_AQUA_LAUNCH=1 where the launch router has code.
func TestLiveTracker(t *testing.T) {
	rpcURL := liveEnv(t, "ONEINCH_AQUA_RPC_URL")[0]
	chain := newScanChain(t, rpcURL)
	ctx, client, head, start := chain.ctx, chain.client, chain.head, chain.start
	cfg := chain.config()

	// 1. One scan of the window, then the same window through GetNewPools.
	lister, err := NewPoolsListUpdater(&cfg, client)
	require.NoError(t, err)
	started := time.Now()
	res := chain.scanLogs(t, lister)
	t.Logf("scan 1: blocks %d..%d in %s, %d strategies shipped", start, head, time.Since(started).Round(time.Second), len(res.shipped))
	_, stats := lister.admitAll(res.shipped, res.docked)
	t.Logf("discards: %v", stats)

	started = time.Now()
	pools, meta, err := lister.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	var m Metadata
	require.NoError(t, json.Unmarshal(meta, &m))
	t.Logf("scan 2 (GetNewPools): up to block %d in %s, listed %d pools", m.LastScannedBlock, time.Since(started).Round(time.Second), len(pools))
	for _, p := range pools {
		require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		require.Len(t, p.Tokens, 2)
		for _, tk := range p.Tokens {
			require.Equal(t, hexutil.Encode(common.HexToAddress(tk.Address).Bytes()), tk.Address, "lowercase")
			require.True(t, tk.Swappable)
		}
	}

	// 2. Track every listed pool in batches of the size the indexer uses.
	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)
	batchSize := int(envUint(t, "ONEINCH_AQUA_LIVE_BATCH", 50))
	var tracked []entity.Pool
	var blocks []uint64
	for _, batch := range lo.Chunk(pools, batchSize) {
		out, errs := tracker.GetNewPoolStates(ctx, batch)
		for i := range out {
			require.NoError(t, errs[i], batch[i].Address)
			tracked = append(tracked, out[i])
		}
		blocks = append(blocks, out[0].BlockNumber)
	}
	var live, docked, inactive, quoted int
	for _, p := range tracked {
		require.Positive(t, p.BlockNumber)
		var x Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &x))
		require.Equal(t, x.RawBlock, x.WalletBlock)
		require.Equal(t, p.BlockNumber, x.RawBlock)
		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		line := map[string]any{"pool": p.Address, "block": p.BlockNumber, "reserves": p.Reserves, "timestamp": p.Timestamp,
			"docked": x.Docked, "inactive": x.Inactive}
		switch {
		case x.Docked:
			docked++
			require.Equal(t, int64(1), p.Timestamp)
			require.Nil(t, sim.CalculateLimit())
		case x.Inactive:
			inactive++
			require.Equal(t, int64(1), p.Timestamp)
		default:
			live++
			var sizes []string
			for in := range 2 {
				for _, num := range []int64{1000, 1_000_000} { // 1/1000 and 1/1e6 of the in-side balance, at least 1 wei
					amt := new(uint256.Int).Div(&sim.State.Raw[in], uint256.NewInt(uint64(num)))
					if amt.IsZero() {
						amt.SetUint64(1)
					}
					r, err := quoteOut(sim, in, amt, nil)
					if err == nil {
						quoted++
						sizes = append(sizes, fmt.Sprintf("%d:%s->%s", in, amt.Dec(), r.TokenAmountOut.Amount))
					} else {
						sizes = append(sizes, fmt.Sprintf("%d:%s->%v", in, amt.Dec(), err))
					}
				}
			}
			line["quotes"] = sizes
		}
		b, _ := json.Marshal(line)
		t.Logf("%s", b)
	}
	t.Logf("TRACKER block=%d pools=%d live=%d docked=%d inactive=%d quotes_ok=%d batches=%d", blocks[len(blocks)-1], len(tracked), live, docked, inactive, quoted, len(blocks))
	require.Positive(t, tracked[0].BlockNumber)
}

// liveEnv skips the test in CI and unless every variable is set, and returns their values.
func liveEnv(t *testing.T, names ...string) []string {
	t.Helper()
	test.SkipCI(t)
	vals := make([]string, len(names))
	for i, name := range names {
		if vals[i] = os.Getenv(name); vals[i] == "" {
			t.Skip("set " + strings.Join(names, ", "))
		}
	}
	return vals
}

// scanChain is the node connection and log window the chain-scanning live tests share. Env: ONEINCH_AQUA_LIVE_WINDOW
// (largest eth_getLogs range, default 50000), ONEINCH_AQUA_LIVE_START (first block) or ONEINCH_AQUA_LIVE_BLOCKS
// (how far back from head, default 150000), ONEINCH_AQUA_LAUNCH=1 where the launch router has code.
type scanChain struct {
	ctx                 context.Context
	client              *ethrpc.Client
	chainId             valueobject.ChainID
	head, window, start uint64
}

func newScanChain(t *testing.T, rpcURL string) scanChain {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Hour) // a full scan of a fast chain from its start block is slow
	t.Cleanup(cancel)
	client := ethrpc.New(rpcURL).SetMulticallContract(multicall3)
	chainId, err := client.GetETHClient().ChainID(ctx)
	require.NoError(t, err)
	head, err := client.GetBlockNumber(ctx)
	require.NoError(t, err)
	l := scanChain{ctx: ctx, client: client, chainId: valueobject.ChainID(chainId.Int64()), head: head,
		window: envUint(t, "ONEINCH_AQUA_LIVE_WINDOW", 50_000), start: envUint(t, "ONEINCH_AQUA_LIVE_START", 0)}
	if l.start == 0 {
		l.start = head - envUint(t, "ONEINCH_AQUA_LIVE_BLOCKS", 150_000)
	}
	return l
}

// config is a lister config that scans the whole window in one GetNewPools call.
func (l scanChain) config(mutate ...func(*Config)) Config {
	cfg := *testConfig(func(c *Config) {
		c.ChainId, c.StartBlock, c.LogWindow, c.MaxWindowsPerRun = l.chainId, l.start, l.window, int((l.head-l.start)/l.window)+2
		if os.Getenv("ONEINCH_AQUA_LAUNCH") == "1" {
			c.LaunchRouterAddress = testLaunchHex
		}
	})
	for _, m := range mutate {
		m(&cfg)
	}
	return cfg
}

// scanLogs reads the log window with the lister's own log reader.
func (l scanChain) scanLogs(t *testing.T, lister *PoolsListUpdater) scanResult {
	t.Helper()
	res := scanResult{docked: map[strategyKey]struct{}{}}
	for from := l.start; from <= l.head; from += l.window {
		var (
			logs    []types.Log
			skipped uint64
		)
		retryLive(t, "eth_getLogs", func() (err error) {
			logs, skipped, err = lister.getLogs(l.ctx, from, min(from+l.window-1, l.head), l.head)
			return err
		})
		res.skipped += skipped
		lister.collect(&res, logs)
	}
	return res
}

// allPools runs GetNewPools until the cursor reaches the confirmed head, retrying a failed run: a run that
// loses a later window returns what it has and the cursor where it stopped, so one call may not be the whole scan.
func (l scanChain) allPools(t *testing.T, lister *PoolsListUpdater, confirmations uint64) []entity.Pool {
	t.Helper()
	var (
		pools []entity.Pool
		meta  []byte
	)
	for {
		var (
			got  []entity.Pool
			next []byte
		)
		retryLive(t, "GetNewPools", func() (err error) {
			got, next, err = lister.GetNewPools(l.ctx, meta)
			return err
		})
		pools, meta = mergePools(pools, got), next
		if metaOf(t, meta).LastScannedBlock+confirmations >= l.head {
			return pools
		}
	}
}

// retryLive runs a call against a public node, which now and then drops a connection: up to 20 tries.
func retryLive(t *testing.T, what string, call func() error) {
	t.Helper()
	var err error
	for attempt := 1; attempt <= 20; attempt++ {
		if err = call(); err == nil {
			return
		}
		t.Logf("%s failed (try %d): %v", what, attempt, err)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
	require.NoError(t, err, what)
}

func envUint(t *testing.T, name string, def uint64) uint64 {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	require.NoError(t, err, name)
	return n
}

var _ pool.IPoolSimulator = (*PoolSimulator)(nil)

// parityEntry is one live strategy as shipped: the bytes of its Shipped event.
type parityEntry struct {
	ChainId      int      `json:"chainId"`
	Maker        string   `json:"maker"`
	App          string   `json:"app"`
	StrategyHash string   `json:"strategyHash"`
	Strategy     string   `json:"strategy"`
	Tokens       []string `json:"tokens"`
}

// goldenVector is one recorded router quote with the strategy balances it was taken at.
type goldenVector struct {
	ChainId   int      `json:"chainId"`
	Block     uint64   `json:"block"`
	App       string   `json:"app"`
	Strategy  string   `json:"strategy"`
	Tokens    []string `json:"tokens"`
	Raw       []string `json:"raw"`
	In        int      `json:"in"`
	AmountIn  string   `json:"amountIn"`
	AmountOut string   `json:"amountOut"`
}

const parityTakerTraits = "0x" + "0000000000000000000000000000000000000000" + "0001" // exact in, no threshold

// TestLiveParity compares the interpreter with the router's quote on live strategies at one block.
//
// Env: ONEINCH_AQUA_RPC (archive node of the chain), ONEINCH_AQUA_PARITY_FILE (JSON list of parityEntry),
// optional ONEINCH_AQUA_PARITY_BLOCK, ONEINCH_AQUA_GOLDEN_OUT (writes a sample of vectors) and
// ONEINCH_AQUA_PARITY_REPORT (writes the summary).
func TestLiveParity(t *testing.T) {
	env := liveEnv(t, "ONEINCH_AQUA_RPC", "ONEINCH_AQUA_PARITY_FILE")
	rpcURL, file := env[0], env[1]
	raw, err := os.ReadFile(file)
	require.NoError(t, err)
	var entries []parityEntry
	require.NoError(t, json.Unmarshal(raw, &entries))

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	cl, err := rpc.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer cl.Close()

	var head hexutil.Uint64
	require.NoError(t, cl.CallContext(ctx, &head, "eth_blockNumber"))
	block := uint64(head) - 8
	if v := os.Getenv("ONEINCH_AQUA_PARITY_BLOCK"); v != "" {
		_, err := fmt.Sscan(v, &block)
		require.NoError(t, err)
	}
	blk := hexutil.EncodeUint64(block)

	report := struct {
		ChainId    int            `json:"chainId"`
		Block      uint64         `json:"block"`
		Strategies int            `json:"strategies"`
		Admitted   int            `json:"admitted"`
		Discarded  map[string]int `json:"discarded"`
		Inactive   int            `json:"inactive"`
		Quotes     int            `json:"quotes"`
		Reverts    int            `json:"onchainReverts"`
		Mismatches []string       `json:"mismatches"`
		Shapes     map[string]int `json:"shapes"`
	}{Block: block, Strategies: len(entries), Discarded: map[string]int{}, Shapes: map[string]int{}}
	if len(entries) > 0 {
		report.ChainId = entries[0].ChainId
	}

	type live struct {
		e      parityEntry
		order  *Order
		prog   []Instruction
		tokens [2]common.Address
		raw    [2]uint256.Int
	}
	var lives []live
	for _, e := range entries {
		rawStrategy, err := hexutil.Decode(e.Strategy)
		require.NoError(t, err)
		order, err := DecodeOrder(rawStrategy)
		if err != nil || order.Hash() != common.HexToHash(e.StrategyHash) || order.Maker != common.HexToAddress(e.Maker) {
			report.Discarded[string(ReasonNonCanonical)]++
			continue
		}
		prog, err := ParseOrder(order, strings.EqualFold(e.App, testLaunchHex))
		if err != nil {
			report.Discarded[string(ReasonOf(err))]++
			continue
		}
		if len(e.Tokens) != 2 {
			report.Discarded["token_count"]++
			continue
		}
		lives = append(lives, live{e: e, order: order, prog: prog,
			tokens: [2]common.Address{common.HexToAddress(e.Tokens[0]), common.HexToAddress(e.Tokens[1])}})
	}

	// rawBalances of both tokens at the pinned block.
	const batch = 50
	var admitted []live
	for lo := 0; lo < len(lives); lo += batch {
		hi := min(lo+batch, len(lives))
		elems := make([]rpc.BatchElem, 0, 2*(hi-lo))
		outs := make([]hexutil.Bytes, 2*(hi-lo))
		for i := lo; i < hi; i++ {
			for k := range 2 {
				data, err := aquaABI.Pack("rawBalances", lives[i].order.Maker, common.HexToAddress(lives[i].e.App),
					common.HexToHash(lives[i].e.StrategyHash), lives[i].tokens[k])
				require.NoError(t, err)
				elems = append(elems, rpc.BatchElem{Method: "eth_call", Args: []any{
					map[string]any{"to": testAquaHex, "data": hexutil.Bytes(data)}, blk}, Result: &outs[2*(i-lo)+k]})
			}
		}
		batchCall(ctx, t, cl, elems)
		for i := lo; i < hi; i++ {
			l := lives[i]
			ok := true
			for k := range 2 {
				el := elems[2*(i-lo)+k]
				require.NoError(t, el.Error)
				res := outs[2*(i-lo)+k]
				require.Len(t, res, 64)
				l.raw[k].SetBytes32(res[:32])
				if new(uint256.Int).SetBytes32(res[32:]).Uint64() != 2 {
					ok = false
				}
			}
			if !ok {
				report.Inactive++
				continue
			}
			admitted = append(admitted, l)
		}
	}
	report.Admitted = len(admitted)

	type probe struct {
		l        live
		in       int
		amountIn uint256.Int
		want     quoteResult
		wantErr  error
	}
	var probes []probe
	for _, l := range admitted {
		report.Shapes[shapeOf(l.prog)]++
		for in := range 2 {
			out := 1 - in
			lt := l.tokens[in].Cmp(l.tokens[out]) < 0
			for _, a := range probeAmounts(l.prog, &l.raw[in], &l.raw[out], lt) {
				amt := a
				res, err := runQuote(l.prog, &l.raw[in], &l.raw[out], &amt, lt)
				if err == nil && res.AmountOut.IsZero() {
					err = ErrZeroAmountOut
				}
				probes = append(probes, probe{l: l, in: in, amountIn: amt, want: res, wantErr: err})
			}
		}
	}

	var golden []goldenVector
	goldenSeen := map[string]int{}
	for lo := 0; lo < len(probes); lo += batch {
		hi := min(lo+batch, len(probes))
		elems := make([]rpc.BatchElem, hi-lo)
		outs := make([]hexutil.Bytes, hi-lo)
		for i := lo; i < hi; i++ {
			p := probes[i]
			data, err := routerABI.Pack("quote", struct {
				Maker  common.Address
				Traits *big.Int
				Data   []byte
			}{p.l.order.Maker, p.l.order.Traits.ToBig(), p.l.order.Data}, p.l.tokens[p.in], p.l.tokens[1-p.in],
				p.amountIn.ToBig(), hexutil.MustDecode(parityTakerTraits))
			require.NoError(t, err)
			elems[i-lo] = rpc.BatchElem{Method: "eth_call", Args: []any{map[string]any{
				"from": "0x00000000000000000000000000000000000dEaDb", "to": p.l.e.App, "data": hexutil.Bytes(data)}, blk},
				Result: &outs[i-lo]}
		}
		batchCall(ctx, t, cl, elems)
		for i := lo; i < hi; i++ {
			p, el := probes[i], elems[i-lo]
			report.Quotes++
			if el.Error != nil {
				if _, isRPC := el.Error.(rpc.Error); !isRPC {
					require.NoError(t, el.Error)
				}
				report.Reverts++
				if p.wantErr == nil {
					report.Mismatches = append(report.Mismatches, fmt.Sprintf("%s in=%d amount=%s: router reverts (%v), interpreter quotes %s",
						p.l.e.StrategyHash, p.in, &p.amountIn, el.Error, &p.want.AmountOut))
				}
				continue
			}
			res := outs[i-lo]
			if len(res) != 96 {
				report.Mismatches = append(report.Mismatches, fmt.Sprintf("%s: bad quote return length %d", p.l.e.StrategyHash, len(res)))
				continue
			}
			gotIn, gotOut := new(uint256.Int).SetBytes32(res[:32]), new(uint256.Int).SetBytes32(res[32:64])
			switch {
			case p.wantErr != nil:
				report.Mismatches = append(report.Mismatches, fmt.Sprintf("%s in=%d amount=%s: router quotes %s, interpreter errors (%v)",
					p.l.e.StrategyHash, p.in, &p.amountIn, gotOut, p.wantErr))
			case !gotOut.Eq(&p.want.AmountOut) || !gotIn.Eq(&p.want.AmountIn):
				report.Mismatches = append(report.Mismatches, fmt.Sprintf("%s in=%d amount=%s: router (%s,%s) interpreter (%s,%s)",
					p.l.e.StrategyHash, p.in, &p.amountIn, gotIn, gotOut, &p.want.AmountIn, &p.want.AmountOut))
			default:
				if goldenSeen[shapeOf(p.l.prog)] < 6 && !gotOut.IsZero() {
					goldenSeen[shapeOf(p.l.prog)]++
					golden = append(golden, goldenVector{ChainId: p.l.e.ChainId, Block: block, App: p.l.e.App, Strategy: p.l.e.Strategy,
						Tokens: p.l.e.Tokens, Raw: []string{p.l.raw[0].Dec(), p.l.raw[1].Dec()}, In: p.in,
						AmountIn: p.amountIn.Dec(), AmountOut: gotOut.Dec()})
				}
			}
		}
	}

	if path := os.Getenv("ONEINCH_AQUA_GOLDEN_OUT"); path != "" {
		b, err := json.Marshal(golden)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, b, 0o644))
	}
	if path := os.Getenv("ONEINCH_AQUA_PARITY_REPORT"); path != "" {
		b, err := json.MarshalIndent(report, "", " ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, b, 0o644))
	}
	t.Logf("chain %d block %d: %d strategies, %d admitted, %d quotes, %d on-chain reverts, %d mismatches, discarded %v",
		report.ChainId, block, report.Strategies, report.Admitted, report.Quotes, report.Reverts, len(report.Mismatches), report.Discarded)
	require.Empty(t, report.Mismatches[:min(len(report.Mismatches), 10)])
}

func shapeOf(prog []Instruction) string {
	names := make([]string, len(prog))
	for i, ins := range prog {
		names[i] = map[uint8]string{opXYCSwap: "xyc", opXYCConcentrate: "concentrate", opSalt: "salt",
			opFlatFeeIn: "flat", opAquaProtocolFee: "protocolFee"}[ins.Op]
	}
	return strings.Join(names, ",")
}

// probeAmounts are amounts from dust to 90% of the output balance, plus off-grid values.
func probeAmounts(prog []Instruction, balIn, balOut *uint256.Int, tokenInLt bool) []uint256.Int {
	fixed := []string{"1", "2", "10", "12345", "1000007", "1000000003", "1000000000001", "1000000000000009",
		"1000000000000000011", "5000000000000000000", "1000000000000000000013", "700000000000000000000000001"}
	var out []uint256.Int
	for _, s := range fixed {
		out = append(out, *uint256.MustFromDecimal(s))
	}
	for _, num := range []uint64{1, 10, 50, 90} {
		var target uint256.Int
		target.MulDivOverflow(balOut, uint256.NewInt(num), uint256.NewInt(100))
		if a, ok := amountInFor(prog, balIn, balOut, tokenInLt, &target); ok {
			out = append(out, *a)
		}
	}
	return out
}

// amountInFor finds a small amountIn whose quote reaches target, by bisection on the monotone quote.
func amountInFor(prog []Instruction, balIn, balOut *uint256.Int, tokenInLt bool, target *uint256.Int) (*uint256.Int, bool) {
	if target.IsZero() {
		return nil, false
	}
	lo, hi := uint256.NewInt(1), new(uint256.Int).Lsh(uint256.NewInt(1), 160)
	reach := func(a *uint256.Int) bool {
		r, err := runQuote(prog, balIn, balOut, a, tokenInLt)
		return err == nil && !r.AmountOut.Lt(target)
	}
	if !reach(hi) {
		return nil, false
	}
	for lo.Lt(hi) {
		mid := new(uint256.Int).Add(lo, hi)
		mid.Rsh(mid, 1)
		if reach(mid) {
			hi = mid
		} else {
			lo = mid.AddUint64(mid, 1)
		}
	}
	return hi, true
}

// batchCall sends a JSON-RPC batch and retries the whole batch on rate limits.
func batchCall(ctx context.Context, t *testing.T, cl *rpc.Client, elems []rpc.BatchElem) {
	t.Helper()
	var err error
	for attempt := range 8 {
		if err = cl.BatchCallContext(ctx, elems); err == nil {
			limited := false
			for _, el := range elems {
				if el.Error != nil && strings.Contains(el.Error.Error(), "429") {
					limited = true
				}
			}
			if !limited {
				return
			}
		}
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	require.NoError(t, err)
	t.Fatal("rate limited after retries")
}

// TestLivePortalVsLogs compares the pool ids and the pools of the Portal path and the log path on one chain.
//
// Env: ONEINCH_AQUA_PORTAL_KEY (the Bearer key, never printed), ONEINCH_AQUA_RPC_URL (a node with eth_getLogs
// ranges of at least ONEINCH_AQUA_LIVE_WINDOW blocks, default 50000), ONEINCH_AQUA_LIVE_BLOCKS (how far back
// the log scan goes, default 150000) or ONEINCH_AQUA_LIVE_START, optional ONEINCH_AQUA_PORTAL_URL (default
// https://api.1inch.com) and ONEINCH_AQUA_LAUNCH=1 where the launch router has code. Run it once per chain.
//
// Checks: the pools both
// paths list are byte-identical (ids, tokens in ascending order, static extra); a pool only the Portal lists
// is older than the scan window; a pool only the logs list is docked or not shipped on chain at head (the
// Portal lags docks and ships), with a small tolerance for the newest ships.
func TestLivePortalVsLogs(t *testing.T) {
	env := liveEnv(t, "ONEINCH_AQUA_PORTAL_KEY", "ONEINCH_AQUA_RPC_URL")
	key := env[0]
	portalURL := os.Getenv("ONEINCH_AQUA_PORTAL_URL")
	if portalURL == "" {
		portalURL = "https://api.1inch.com"
	}
	chain := newScanChain(t, env[1])
	ctx, client, head, start := chain.ctx, chain.client, chain.head, chain.start
	retries := 3
	confirmations := envUint(t, "ONEINCH_AQUA_LIVE_CONFIRMATIONS", 12)
	cfg := chain.config(func(c *Config) {
		c.DexId, c.Confirmations = DexType, &confirmations
		c.HTTPConfig.BaseURL, c.HTTPConfig.ApiKey, c.PortalPageLimit, c.PortalMaxPages = portalURL, key, 100, 200
		c.HTTPConfig.Timeout, c.HTTPConfig.RetryCount, c.HTTPConfig.RetryWaitTime = durationjson.Duration{Duration: 30 * time.Second}, &retries, durationjson.Duration{Duration: 2 * time.Second}
	})

	probe, err := NewPoolsListUpdater(&cfg, client)
	require.NoError(t, err)
	require.NotNil(t, probe.portal, "the Portal must be enabled")
	res := chain.scanLogs(t, probe)
	sweepStart := time.Now()
	items, perr := probe.portal.sweep(ctx)
	sweepTime := time.Since(sweepStart)
	require.Nil(t, perr, "the Portal sweep failed")
	_, bad := probe.portalShipped(items)
	require.Zero(t, bad, "items the lister cannot read")

	logCfg := cfg
	logCfg.HTTPConfig.BaseURL = ""
	logLister, err := NewPoolsListUpdater(&logCfg, client)
	require.NoError(t, err)
	logPools := chain.allPools(t, logLister, confirmations)
	portalLister, err := NewPoolsListUpdater(&cfg, client)
	require.NoError(t, err)
	portalPools, meta, err := portalLister.GetNewPools(ctx, nil)
	require.NoError(t, err)
	// A first full sweep stores only the confirmed head (read before the sweep, so at or after the one seen at
	// the start: a live chain moves) and leaves the cursor at 0.
	stored := metaOf(t, meta)
	require.Zero(t, stored.LastScannedBlock, "a first full sweep does not move the cursor")
	require.GreaterOrEqual(t, stored.LastPortalHead, head-confirmations)

	lm, pm := idsOf(logPools), idsOf(portalPools)
	var both, portalOnly, logOnly int
	var logOnlyPools []entity.Pool
	for id, lp := range lm {
		pp, ok := pm[id]
		if !ok {
			logOnly++
			logOnlyPools = append(logOnlyPools, lp)
			continue
		}
		both++
		lp.Timestamp, pp.Timestamp = 0, 0
		require.Equal(t, lp, pp, "pool %s differs between the two paths", id)
	}
	for id := range pm {
		if _, ok := lm[id]; !ok {
			portalOnly++
		}
	}
	t.Logf("chain %d: log pools %d, Portal pools %d, in both %d, Portal only %d (opened before block %d), logs only %d", chain.chainId, len(lm), len(pm), both, portalOnly, start, logOnly)
	t.Logf("PORTALROW chain=%d portal_pools=%d log_pools=%d both=%d portal_only=%d logs_only=%d sweep=%s items=%d", chain.chainId, len(pm), len(lm), both, portalOnly, logOnly, sweepTime.Round(time.Millisecond), len(items))
	// A chain may admit nothing (every strategy discarded by the admission rules): both paths then agree on an
	// empty list, which is a valid result; it must still have strategies at all.
	require.Positive(t, len(items)+len(res.shipped), "no strategy on the chain, in the Portal or in the logs")

	// A pool the logs list and the Portal does not must be docked or not shipped at head.
	if len(logOnlyPools) > 0 {
		tracker, err := NewPoolTracker(&cfg, client)
		require.NoError(t, err)
		states, errs := tracker.GetNewPoolStates(ctx, logOnlyPools)
		var live int
		for i, p := range states {
			require.NoError(t, errs[i])
			if x := decodeExtra(t, p); !x.Docked && !x.Inactive {
				live++
			}
		}
		t.Logf("logs-only pools: %d, still live on chain: %d", len(logOnlyPools), live)
		require.LessOrEqual(t, live, max(2, len(lm)/100), "live strategies the Portal does not list")
	}
}

// countingConn counts the bytes of the TCP connection under TLS: headers, TLS records and the body as
// they cross the wire (compressed when the node compresses).
type countingConn struct {
	net.Conn
	read, written *atomic.Int64
}

func (c *countingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.read.Add(int64(n))
	return n, err
}

func (c *countingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.written.Add(int64(n))
	return n, err
}

// meter counts the traffic of one RPC client. Wire bytes include HTTP headers and TLS framing; body bytes
// are the JSON-RPC payloads before and after decompression.
type meter struct {
	wireIn, wireOut atomic.Int64
	bodyIn, bodyOut atomic.Int64
	requests        atomic.Int64
}

type meterSnapshot struct{ WireIn, WireOut, BodyIn, BodyOut, Requests int64 }

func (m *meter) snapshot() meterSnapshot {
	return meterSnapshot{m.wireIn.Load(), m.wireOut.Load(), m.bodyIn.Load(), m.bodyOut.Load(), m.requests.Load()}
}

func (a meterSnapshot) sub(b meterSnapshot) meterSnapshot {
	return meterSnapshot{a.WireIn - b.WireIn, a.WireOut - b.WireOut, a.BodyIn - b.BodyIn, a.BodyOut - b.BodyOut, a.Requests - b.Requests}
}

func (a meterSnapshot) add(b meterSnapshot) meterSnapshot {
	return meterSnapshot{a.WireIn + b.WireIn, a.WireOut + b.WireOut, a.BodyIn + b.BodyIn, a.BodyOut + b.BodyOut, a.Requests + b.Requests}
}

type bodyCounter struct {
	io.ReadCloser
	n *atomic.Int64
}

func (b *bodyCounter) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.n.Add(int64(n))
	return n, err
}

type meteredTransport struct {
	base *http.Transport
	m    *meter
}

func (t *meteredTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		t.m.bodyOut.Add(int64(len(body)))
		req = req.Clone(req.Context())
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	}
	t.m.requests.Add(1)
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &bodyCounter{ReadCloser: resp.Body, n: &t.m.bodyIn}
	return resp, nil
}

func newMeteredClient(t *testing.T, rawURL string) (*ethrpc.Client, *meter) {
	t.Helper()
	m := &meter{}
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	base := &http.Transport{
		MaxIdleConns: 4, IdleConnTimeout: time.Minute,
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			raw, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			conn := tls.Client(&countingConn{Conn: raw, read: &m.wireIn, written: &m.wireOut}, &tls.Config{ServerName: u.Hostname()})
			if err := conn.HandshakeContext(ctx); err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
	httpClient := &http.Client{Transport: &meteredTransport{base: base, m: m}}
	rc, err := rpc.DialOptions(context.Background(), rawURL, rpc.WithHTTPClient(httpClient))
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return ethrpc.NewWithClient(ethclient.NewClient(rc)).SetMulticallContract(multicall3), m
}

// benchPools builds the pools the lister would, from a file of shipped strategies.
func benchPools(t *testing.T, file string) []entity.Pool {
	t.Helper()
	raw, err := os.ReadFile(file)
	require.NoError(t, err)
	var entries []parityEntry
	require.NoError(t, json.Unmarshal(raw, &entries))
	u := &PoolsListUpdater{cfg: &settings{dexId: DexType, aqua: testAqua, router: testRouter, launch: testLaunch, hasLaunch: true}}
	var pools []entity.Pool
	for _, e := range entries {
		rawStrategy, err := hexutil.Decode(e.Strategy)
		require.NoError(t, err)
		s := &shipped{maker: common.HexToAddress(e.Maker), app: common.HexToAddress(e.App), hash: common.HexToHash(e.StrategyHash), strategy: rawStrategy}
		for _, tk := range e.Tokens {
			s.tokens = append(s.tokens, common.HexToAddress(tk))
		}
		if p, reason := u.admit(s, nil); reason == "" {
			pools = append(pools, *p)
		}
	}
	return pools
}

// TestRPCBytesBenchmark measures the request and response bytes of one refresh of a chain's strategies,
// for the full refresh the tracker does and for the wallet reads alone (the lower bound without the raw
// balance reads), at the indexer batch size, over real connections.
//
// Env: ONEINCH_AQUA_RPC_URL, ONEINCH_AQUA_PARITY_FILE, optional ONEINCH_AQUA_BENCH_BATCH (default 50),
// ONEINCH_AQUA_BENCH_REPEAT (default 3), ONEINCH_AQUA_BENCH_OUT (JSON result path).
func TestRPCBytesBenchmark(t *testing.T) {
	env := liveEnv(t, "ONEINCH_AQUA_RPC_URL", "ONEINCH_AQUA_PARITY_FILE")
	rpcURL, file := env[0], env[1]
	pools := benchPools(t, file)
	require.NotEmpty(t, pools)
	batchSize := int(envUint(t, "ONEINCH_AQUA_BENCH_BATCH", 50))
	repeat := int(envUint(t, "ONEINCH_AQUA_BENCH_REPEAT", 3))
	client, m := newMeteredClient(t, rpcURL)
	tracker, err := NewPoolTracker(testConfig(), client)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	batches := lo.Chunk(pools, batchSize)
	// Unique (maker, token) pairs per batch: the wallet reads of the lower bound.
	walletOnly := func() {
		for _, batch := range batches {
			req := client.NewRequest().SetContext(ctx)
			seen := map[[2]string]bool{}
			for _, p := range batch {
				var st StaticExtra
				require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &st))
				for _, tk := range p.Tokens {
					key := [2]string{st.Maker, tk.Address}
					if seen[key] {
						continue
					}
					seen[key] = true
					var b, a *big.Int
					req.AddCall(&ethrpc.Call{ABI: erc20ABI, Target: tk.Address, Method: "balanceOf", Params: []any{common.HexToAddress(st.Maker)}}, []any{&b})
					req.AddCall(&ethrpc.Call{ABI: erc20ABI, Target: tk.Address, Method: "allowance", Params: []any{common.HexToAddress(st.Maker), testAqua}}, []any{&a})
				}
			}
			_, err := req.TryBlockAndAggregate()
			require.NoError(t, err)
		}
	}
	full := func() {
		for _, batch := range batches {
			_, errs := tracker.GetNewPoolStates(ctx, batch)
			for _, e := range errs {
				require.NoError(t, e)
			}
		}
	}

	measure := func(name string, f func()) meterSnapshot {
		f() // warm the connection
		var total meterSnapshot
		for range repeat {
			before := m.snapshot()
			f()
			total = total.add(m.snapshot().sub(before))
		}
		avg := meterSnapshot{total.WireIn / int64(repeat), total.WireOut / int64(repeat), total.BodyIn / int64(repeat),
			total.BodyOut / int64(repeat), total.Requests / int64(repeat)}
		t.Logf("%s: %+v per refresh of %d pools in %d batches", name, avg, len(pools), len(batches))
		return avg
	}
	fullBytes := measure("full", full)
	walletBytes := measure("wallet_only", walletOnly)

	const usdPerGiB, cycleSeconds = 0.085, 12.0
	usdPerDay := func(bytes int64) float64 { return float64(bytes) * (86400 / cycleSeconds) * usdPerGiB / (1 << 30) }
	result := map[string]any{
		"pools": len(pools), "batchSize": batchSize, "batches": len(batches), "cycleSeconds": cycleSeconds, "usdPerGiB": usdPerGiB,
		"full": benchRow(fullBytes, usdPerDay), "walletOnly": benchRow(walletBytes, usdPerDay),
	}
	b, _ := json.MarshalIndent(result, "", " ")
	t.Logf("%s", b)
	if path := os.Getenv("ONEINCH_AQUA_BENCH_OUT"); path != "" {
		require.NoError(t, os.WriteFile(path, b, 0o644))
	}
}

func benchRow(s meterSnapshot, usdPerDay func(int64) float64) map[string]any {
	return map[string]any{
		"requests":         s.Requests,
		"wireRequestBytes": s.WireOut, "wireResponseBytes": s.WireIn,
		"bodyRequestBytes": s.BodyOut, "bodyResponseBytes": s.BodyIn,
		"usdPerDayRequestOnlyWire": usdPerDay(s.WireOut), "usdPerDayBothWire": usdPerDay(s.WireOut + s.WireIn),
		"usdPerDayRequestOnlyBody": usdPerDay(s.BodyOut), "usdPerDayBothBody": usdPerDay(s.BodyOut + s.BodyIn),
	}
}

var pulledTopic = crypto.Keccak256Hash([]byte("Pulled(address,address,bytes32,address,uint256)"))

// replayFile is the input of the replay test (ONEINCH_AQUA_REPLAY_*): one strategy and the real swap
// transactions on it, oldest first.
type replayFile struct {
	ChainId  int `json:"chainId"`
	Strategy struct {
		parityEntry
		Block uint64 `json:"block"`
	} `json:"strategy"`
	ForkBlock uint64 `json:"forkBlock"`
	Taker     string `json:"taker"`
	// Fund maps a token to the state writes that give the taker its balance.
	Fund  map[string]fundSpec `json:"fund"`
	Swaps []struct {
		Tx       string      `json:"tx"`
		Block    uint64      `json:"block"`
		TokenIn  string      `json:"tokenIn"`
		AmountIn json.Number `json:"amountIn"`
		TokenOut string      `json:"tokenOut"`
	} `json:"swaps"`
}

// fundSpec is the state writes that fund an account: storage slots and native balances.
type fundSpec struct {
	Wallet    string                       `json:"wallet"`
	Strategy  string                       `json:"strategy"`
	StateDiff map[string]map[string]string `json:"stateDiff"`
	Balance   map[string]string            `json:"balance"`
	Refused   string                       `json:"refused"`
}

type replayRow struct {
	Tx           string `json:"tx"`
	OrigBlock    uint64 `json:"origBlock"`
	ForkBlock    uint64 `json:"forkBlock"`
	Mode         string `json:"mode"`
	Direction    string `json:"direction"`
	AmountIn     string `json:"amountIn"`
	SimOut       string `json:"simOut"`
	ObservedOut  string `json:"observedOut"`
	SimFee       string `json:"simFeeCollected"`
	ObservedFee  string `json:"observedFeePulled"`
	Gas          uint64 `json:"txGasUsed"`
	StateMatch   bool   `json:"stateMatch"`
	LimitMatch   bool   `json:"limitMatch"`
	ReservesOK   bool   `json:"reservesMatch"`
	Verdict      string `json:"verdict"`
	Detail       string `json:"detail,omitempty"`
	RawAfterSim  string `json:"rawAfterSim"`
	RawAfterReal string `json:"rawAfterChain"`
}

// TestStatefulReplay replays real swaps of one live strategy on an anvil fork, one after the other with no
// revert in between, and after each swap compares the simulator (CalcAmountOut, then UpdateBalance with a
// shared SwapLimit, as the routing backend does) with what the chain did: the amounts in the receipt's Aqua
// events and the state the real tracker reads from the fork (raw balances, maker wallet, allowance).
//
// Env: ONEINCH_AQUA_RPC (archive node, for the lister at the ship block and the original transactions),
// ONEINCH_AQUA_ANVIL (anvil forked at forkBlock with --auto-impersonate), ONEINCH_AQUA_REPLAY_FILE,
// optional ONEINCH_AQUA_REPLAY_OUT (JSON rows).
func TestStatefulReplay(t *testing.T) {
	env := liveEnv(t, "ONEINCH_AQUA_RPC", "ONEINCH_AQUA_ANVIL", "ONEINCH_AQUA_REPLAY_FILE")
	archiveURL, anvilURL, file := env[0], env[1], env[2]
	raw, err := os.ReadFile(file)
	require.NoError(t, err)
	var rf replayFile
	require.NoError(t, json.Unmarshal(raw, &rf))
	require.NotEmpty(t, rf.Swaps)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	archive := ethrpc.New(archiveURL).SetMulticallContract(multicall3)
	fork := ethrpc.New(anvilURL).SetMulticallContract(multicall3)
	archiveRPC, err := rpc.DialContext(ctx, archiveURL)
	require.NoError(t, err)
	defer archiveRPC.Close()
	forkRPC, err := rpc.DialContext(ctx, anvilURL)
	require.NoError(t, err)
	defer forkRPC.Close()

	var forkHead hexutil.Uint64
	require.NoError(t, forkRPC.CallContext(ctx, &forkHead, "eth_blockNumber"))
	require.Equal(t, rf.ForkBlock, uint64(forkHead), "the anvil fork must sit at forkBlock")

	// The real lister at the block the strategy shipped in builds the pool.
	cfg := *testConfig(func(c *Config) {
		c.ChainId = valueobject.ChainID(rf.ChainId)
		c.StartBlock, c.LogWindow, c.MaxWindowsPerRun = 1, 1000, 1
		if strings.EqualFold(rf.Strategy.App, testLaunchHex) {
			c.LaunchRouterAddress = testLaunchHex
		}
	})
	lister, err := NewPoolsListUpdater(&cfg, archive)
	require.NoError(t, err)
	shipBlock := rf.Strategy.Block
	require.NotZero(t, shipBlock, "strategy block missing in the replay file")
	res := scanResult{docked: map[strategyKey]struct{}{}}
	logs, _, err := lister.getLogs(ctx, shipBlock, shipBlock, shipBlock+1_000_000)
	require.NoError(t, err)
	lister.collect(&res, logs)
	pools, stats := lister.admitAll(res.shipped, res.docked)
	var listed *entity.Pool
	for i := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(pools[i].StaticExtra), &se))
		if strings.EqualFold(se.StrategyHash, rf.Strategy.StrategyHash) {
			listed = &pools[i]
		}
	}
	require.NotNil(t, listed, "the lister did not list the strategy (discards %v)", stats)
	t.Logf("lister at block %d listed pool %s (%s %s)", shipBlock, listed.Address, listed.Tokens[0].Address, listed.Tokens[1].Address)

	tracker, err := NewPoolTracker(&cfg, fork)
	require.NoError(t, err)
	track := func() (entity.Pool, Extra) {
		out, errs := tracker.GetNewPoolStates(ctx, []entity.Pool{*listed})
		require.NoError(t, errs[0])
		var x Extra
		require.NoError(t, json.Unmarshal([]byte(out[0].Extra), &x))
		return out[0], x
	}
	p0, x0 := track()
	require.False(t, x0.Docked || x0.Inactive, "strategy must be live at the fork block")
	sim, err := NewPoolSimulator(p0)
	require.NoError(t, err)
	t.Logf("start: fork block %d tracker block %d raw %s/%s wallet %s/%s", rf.ForkBlock, p0.BlockNumber, x0.RawBalance0.Dec(), x0.RawBalance1.Dec(),
		x0.WalletBalance0.Dec(), x0.WalletBalance1.Dec())
	limits := swaplimit.NewInventory(DexType, sim.CalculateLimit())
	hashTopic := common.HexToHash(rf.Strategy.StrategyHash)

	var rows []replayRow
	passed := 0
	for _, sw := range rf.Swaps {
		row := replayRow{Tx: sw.Tx, OrigBlock: sw.Block}
		// 1. The original transaction, impersonated. It settles when everything it depends on is still the
		// same on the fork; a bot transaction with other legs or a price check often is not.
		var o map[string]any
		require.NoError(t, archiveRPC.CallContext(ctx, &o, "eth_getTransactionByHash", sw.Tx))
		origTx := map[string]any{
			"from": str(o["from"]), "to": str(o["to"]), "data": str(o["input"]), "value": str(o["value"]), "gas": str(o["gas"])}
		// The original sender gets gas money and the call is estimated first, so a transaction that cannot settle on the
		// fork is never queued (a queued transaction could be mined later and move the strategy behind the simulator).
		require.NoError(t, forkRPC.CallContext(ctx, nil, "anvil_setBalance", origTx["from"], "0x3635c9adc5dea00000"))
		var rcpt *types.Receipt
		var err error
		var est hexutil.Uint64
		if err = forkRPC.CallContext(ctx, &est, "eth_estimateGas", origTx); err == nil {
			rcpt, err = send(ctx, forkRPC, origTx)
			require.NoError(t, err, "an estimated transaction must be mined")
		}
		row.Mode = "original transaction"
		if err != nil || rcpt == nil || rcpt.Status != types.ReceiptStatusSuccessful {
			// 2. The same Aqua swap (direction, amount in) through the real router from a funded taker.
			row.Detail = "original transaction does not settle on the fork"
			if err != nil {
				row.Detail += ": " + err.Error()
			}
			var amt big.Int
			_, ok := amt.SetString(sw.AmountIn.String(), 10)
			require.True(t, ok)
			rcpt = directSwap(ctx, t, forkRPC, &rf, sw.TokenIn, sw.TokenOut, &amt)
			row.Mode = "direct router swap, same direction and amount"
		}
		require.NotNil(t, rcpt)
		row.ForkBlock, row.Gas = rcpt.BlockNumber.Uint64(), rcpt.GasUsed
		if rcpt.Status != types.ReceiptStatusSuccessful {
			row.Verdict, row.Detail = "skipped", "the swap reverted on the fork"
			rows = append(rows, row)
			continue
		}
		// What the chain did to this strategy, from the receipt.
		var tokenIn, tokenOut common.Address
		var amountIn, amountOut, feePulled uint256.Int
		var pushes, pulls []struct {
			token  common.Address
			amount uint256.Int
		}
		for _, lg := range rcpt.Logs {
			if lg.Address != testAqua {
				continue
			}
			if len(lg.Topics) != 1 || len(lg.Data) != 160 || common.BytesToHash(lg.Data[64:96]) != hashTopic {
				continue
			}
			e := struct {
				token  common.Address
				amount uint256.Int
			}{common.BytesToAddress(lg.Data[96:128]), *new(uint256.Int).SetBytes32(lg.Data[128:160])}
			switch lg.Topics[0] {
			case pushedEvent.ID:
				pushes = append(pushes, e)
			case pulledTopic:
				pulls = append(pulls, e)
			}
		}
		require.Len(t, pushes, 1, sw.Tx)
		tokenIn, amountIn = pushes[0].token, pushes[0].amount
		for _, pl := range pulls {
			if pl.token == tokenIn {
				feePulled.Add(&feePulled, &pl.amount)
			} else {
				tokenOut, amountOut = pl.token, pl.amount
			}
		}
		require.NotEqual(t, common.Address{}, tokenOut, sw.Tx)
		in := sim.GetTokenIndex(strings.ToLower(tokenIn.Hex()))
		out := sim.GetTokenIndex(strings.ToLower(tokenOut.Hex()))
		require.True(t, in >= 0 && out >= 0 && in != out, sw.Tx)
		row.Direction = fmt.Sprintf("%s->%s", tokenIn.Hex(), tokenOut.Hex())
		row.AmountIn, row.ObservedOut, row.ObservedFee = amountIn.Dec(), amountOut.Dec(), feePulled.Dec()

		// The simulator quotes the same input from its own evolved state, with the shared limit.
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[in], Amount: amountIn.ToBig()},
			TokenOut:      sim.Info.Tokens[out],
			Limit:         limits,
		})
		if err != nil {
			row.Verdict, row.Detail = "FAIL", "simulator refused a swap the chain settled: "+err.Error()
			rows = append(rows, row)
			continue
		}
		info := res.SwapInfo.(*SwapInfo)
		row.SimOut, row.SimFee = res.TokenAmountOut.Amount.String(), info.FeeCollected.Dec()
		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: sim.Info.Tokens[in], Amount: amountIn.ToBig()},
			TokenAmountOut: *res.TokenAmountOut,
			Fee:            *res.Fee,
			SwapInfo:       res.SwapInfo,
			SwapLimit:      limits,
		})

		// The chain's state after the swap, read by the real tracker.
		pn, xn := track()
		row.RawAfterSim = fmt.Sprintf("%s/%s", sim.State.Raw[0].Dec(), sim.State.Raw[1].Dec())
		row.RawAfterReal = fmt.Sprintf("%s/%s", xn.RawBalance0.Dec(), xn.RawBalance1.Dec())
		after := xn.state()
		// An allowance from 2^128 is "unlimited" in the simulator: a token may still lower it at each pull (USDG does), which
		// the simulator does not follow and the next refresh corrects; the limit it publishes is unaffected.
		allowSame := func(a, b *uint256.Int) bool {
			return a.Eq(b) || (!a.Lt(uUnlimitedAllowance) && !b.Lt(uUnlimitedAllowance))
		}
		row.StateMatch = sim.State.Raw == after.Raw && sim.State.Wallet == after.Wallet &&
			allowSame(&sim.State.Allowance[0], &after.Allowance[0]) && allowSame(&sim.State.Allowance[1], &after.Allowance[1])
		if !row.StateMatch {
			row.Detail = fmt.Sprintf("raw sim %s/%s chain %s/%s; wallet sim %s/%s chain %s/%s; allowance sim %s/%s chain %s/%s",
				sim.State.Raw[0].Dec(), sim.State.Raw[1].Dec(), after.Raw[0].Dec(), after.Raw[1].Dec(),
				sim.State.Wallet[0].Dec(), sim.State.Wallet[1].Dec(), after.Wallet[0].Dec(), after.Wallet[1].Dec(),
				sim.State.Allowance[0].Dec(), sim.State.Allowance[1].Dec(), after.Allowance[0].Dec(), after.Allowance[1].Dec())
		}
		chainLimit := [2]uint256.Int{}
		for i := range 2 {
			chainLimit[i] = minU(after.Wallet[i], after.Allowance[i])
		}
		row.LimitMatch = true
		for i := range 2 {
			got := limits.GetLimit(sim.limitKeys[i])
			if got == nil || got.Cmp(chainLimit[i].ToBig()) != 0 {
				row.LimitMatch = false
			}
		}
		row.ReservesOK = pn.Reserves[0] == sim.Info.Reserves[0].String() && pn.Reserves[1] == sim.Info.Reserves[1].String()
		switch {
		case res.TokenAmountOut.Amount.Cmp(amountOut.ToBig()) != 0:
			row.Verdict = "FAIL"
			row.Detail = "simulated amountOut differs from the chain"
		case info.FeeCollected.Cmp(&feePulled) != 0:
			row.Verdict = "FAIL"
			row.Detail = "simulated protocol fee differs from the chain"
		case !row.StateMatch:
			row.Verdict = "FAIL"
			row.Detail = "state after UpdateBalance differs from the tracker's read of the fork: " + row.Detail
		case !row.LimitMatch:
			row.Verdict = "FAIL"
			row.Detail = "SwapLimit after UpdateBalance differs from the maker's wallet limit on the fork"
		case !row.ReservesOK:
			row.Verdict = "FAIL"
			row.Detail = "published reserves differ from the tracker's"
		default:
			row.Verdict = "pass"
			passed++
		}
		rows = append(rows, row)
		// Re-sync the simulator from the tracker for the next row only when it diverged, so one error does
		// not hide the rest; a passing row leaves the simulator on its own evolved state.
		if row.Verdict != "pass" {
			sim, err = NewPoolSimulator(pn)
			require.NoError(t, err)
			limits = swaplimit.NewInventory(DexType, sim.CalculateLimit())
		}
	}
	b, _ := json.MarshalIndent(rows, "", " ")
	t.Logf("ROWS %s", b)
	if out := os.Getenv("ONEINCH_AQUA_REPLAY_OUT"); out != "" {
		require.NoError(t, os.WriteFile(out, b, 0o644))
	}
	var failed int
	for _, r := range rows {
		if r.Verdict == "FAIL" {
			failed++
		}
	}
	t.Logf("REPLAY chain=%d swaps=%d passed=%d failed=%d skipped=%d", rf.ChainId, len(rf.Swaps), passed, failed, len(rows)-passed-failed)
	require.Zero(t, failed)
	require.GreaterOrEqual(t, passed, 3, "at least three swaps must replay sequentially")
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func minU(a, b uint256.Int) uint256.Int {
	if b.Lt(&a) {
		return b
	}
	return a
}

// send mines one impersonated transaction on the fork and returns its receipt.
func send(ctx context.Context, c *rpc.Client, tx map[string]any) (*types.Receipt, error) {
	var hash common.Hash
	if err := c.CallContext(ctx, &hash, "eth_sendTransaction", tx); err != nil {
		return nil, err
	}
	var rcpt *types.Receipt
	for range 50 {
		if err := c.CallContext(ctx, &rcpt, "eth_getTransactionReceipt", hash); err == nil && rcpt != nil {
			return rcpt, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("no receipt for %s", hash)
}

// directSwap funds the taker with amountIn of tokenIn (storage write at the key the driver found), approves the
// strategy's app and calls app.swap(order, tokenIn, tokenOut, amount, takerTraits) exactly as the executor's
// Generic hop does: exact in, useTransferFromAndAquaPush, tokenOut paid to the taker.
func directSwap(ctx context.Context, t *testing.T, c *rpc.Client, rf *replayFile, tokenIn, tokenOut string, amount *big.Int) *types.Receipt {
	t.Helper()
	taker, app := common.HexToAddress(rf.Taker), common.HexToAddress(rf.Strategy.App)
	spec, ok := rf.Fund[strings.ToLower(tokenIn)]
	require.True(t, ok && spec.Refused == "", "no funding for %s: %s", tokenIn, spec.Refused)
	for addr, slots := range spec.StateDiff {
		for slot, val := range slots {
			require.NoError(t, c.CallContext(ctx, nil, "anvil_setStorageAt", addr, slot, val))
		}
	}
	for addr, bal := range spec.Balance {
		require.NoError(t, c.CallContext(ctx, nil, "anvil_setBalance", addr, bal))
	}
	require.NoError(t, c.CallContext(ctx, nil, "anvil_setBalance", taker.Hex(), "0x3635c9adc5dea00000")) // gas money, 1000 native
	_ = amount
	approve, err := swapABI.Pack("approve", app, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)))
	require.NoError(t, err)
	rc, err := send(ctx, c, map[string]any{"from": taker.Hex(), "to": tokenIn, "data": hexutil.Encode(approve), "gas": "0x100000"})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, rc.Status, "approve")

	strat, err := hexutil.Decode(rf.Strategy.Strategy)
	require.NoError(t, err)
	order, err := DecodeOrder(strat)
	require.NoError(t, err)
	var header []byte
	for range 9 {
		header = append(header, 0x00, 0x14)
	}
	header = append(header, 0x00, 0x00, 0x00, 0x41)
	takerTraits := append(header, taker[:]...)
	data, err := swapABI.Pack("swap", struct {
		Maker  common.Address
		Traits *big.Int
		Data   []byte
	}{order.Maker, order.Traits.ToBig(), order.Data}, common.HexToAddress(tokenIn), common.HexToAddress(tokenOut), amount, takerTraits)
	require.NoError(t, err)
	rcpt, err := send(ctx, c, map[string]any{"from": taker.Hex(), "to": app.Hex(), "data": hexutil.Encode(data), "gas": "0x1000000"})
	require.NoError(t, err)
	return rcpt
}

var swapABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"swap","stateMutability":"nonpayable","inputs":[
	 {"name":"order","type":"tuple","components":[{"name":"maker","type":"address"},{"name":"traits","type":"uint256"},{"name":"data","type":"bytes"}]},
	 {"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"amount","type":"uint256"},{"name":"takerTraits","type":"bytes"}],
	 "outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"},{"name":"","type":"bytes32"}]},
	 {"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()
