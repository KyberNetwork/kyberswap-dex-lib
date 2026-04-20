package ambient

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const crocImpactAddr = "0x3e3EDd3eD7621891E574E5d7f47b1f30A994c0D0"

type testHarness struct {
	ctx      context.Context
	client   *ethclient.Client
	tracker  *StateTracker
	rpcURL   string
	blockBI  *big.Int
	blockNum uint64
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	test.SkipCI(t)

	rpcURL := os.Getenv("ETHEREUM_RPC_URL")
	if rpcURL == "" {
		rpcURL = testLTRPCURL
	}
	ctx := t.Context()
	client, err := ethclient.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	blockNum, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	t.Logf("block: %d", blockNum)

	return &testHarness{
		ctx:      ctx,
		client:   client,
		tracker:  NewStateTracker(client, testLTSwapDex),
		rpcURL:   rpcURL,
		blockBI:  new(big.Int).SetUint64(blockNum),
		blockNum: blockNum,
	}
}

// TestTickRangeAnalysis fetches every real PoolIdx=420 pool from ambindexer
// and analyzes tick distribution + liquidity-weighted coverage to recommend
// a TickRange that works well across the whole pool set.
func TestTickRangeAnalysis(t *testing.T) {
	t.Parallel()
	h := newTestHarness(t)

	pairs := fetchIndexerPairs(t, 420)
	t.Logf("fetched %d PoolIdx=420 pools", len(pairs))

	pools := loadPoolStats(t, h, pairs)
	logPerPoolDist(t, pools)

	candidates := []int32{5000, 10000, 20000, 30000, 50000, 75000, 100000, 150000, 200000}
	stats := sweepRanges(pools, candidates)
	logSweep(t, stats, len(pools))

	recommendTickRange(t, pools, stats)
}

// TestTickRangeSwapParity verifies CalcAmountOut across windowed TickRanges
// against full-range simulator and on-chain CrocImpact.
func TestTickRangeSwapParity(t *testing.T) {
	t.Parallel()
	h := newTestHarness(t)

	base := valueobject.AddrZero
	quote := common.HexToAddress(testLTUSDC)

	fullState, err := h.tracker.Load(h.ctx, base, quote, 420, h.blockBI)
	require.NoError(t, err)

	currentTick := GetTickAtSqrtRatio(fullState.Curve.PriceRoot)
	t.Logf("ETH/USDC: %d ticks, currentTick=%d", len(fullState.ActiveTicks), currentTick)

	tickRanges := []int32{10000, 20000, 50000}
	windowedSims := make(map[int32]*PoolSimulator, len(tickRanges))
	wethAddr := common.HexToAddress(testLTNativeTokenAddress)
	for _, tr := range tickRanges {
		window := clampWindow(TickWindow{MinTick: currentTick - tr, MaxTick: currentTick + tr})
		ws, err := h.tracker.LoadWindow(h.ctx, base, quote, 420, h.blockBI, window)
		require.NoError(t, err)
		windowedSims[tr] = buildSimulator(t, ws, wethAddr)
		t.Logf("TickRange=%d: %d ticks", tr, len(ws.ActiveTicks))
	}
	fullSim := buildSimulator(t, fullState, wethAddr)

	usdcAddr := common.HexToAddress(testLTUSDC)
	wei18 := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	wei6 := big.NewInt(1_000_000)
	cases := []struct {
		name              string
		tokenIn, tokenOut string
		amountIn          *big.Int
		isBuy, inBaseQty  bool
	}{
		{"0.01 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Div(wei18, big.NewInt(100)), true, true},
		{"0.1 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Div(wei18, big.NewInt(10)), true, true},
		{"1 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Set(wei18), true, true},
		{"10 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Mul(big.NewInt(10), wei18), true, true},
		{"100 USDC→ETH", usdcAddr.Hex(), wethAddr.Hex(), new(big.Int).Mul(big.NewInt(100), wei6), false, false},
		{"1000 USDC→ETH", usdcAddr.Hex(), wethAddr.Hex(), new(big.Int).Mul(big.NewInt(1000), wei6), false, false},
		{"10000 USDC→ETH", usdcAddr.Hex(), wethAddr.Hex(), new(big.Int).Mul(big.NewInt(10000), wei6), false, false},
	}

	t.Logf("\n=== SWAP PARITY: simulator(full) vs simulator(windowed) ===")
	header := fmt.Sprintf("%-20s %18s", "CASE", "FULL_OUT")
	for _, tr := range tickRanges {
		header += fmt.Sprintf(" %14s", fmt.Sprintf("TR=%d", tr))
	}
	t.Log(header)
	for _, tc := range cases {
		fullResult, fullErr := fullSim.CloneState().(*PoolSimulator).
			CalcAmountOut(calcParams(tc.tokenIn, tc.tokenOut, tc.amountIn))
		line := fmt.Sprintf("%-20s", tc.name)
		if fullErr != nil {
			line += fmt.Sprintf(" %18s", "ERR:"+fullErr.Error())
		} else {
			line += fmt.Sprintf(" %18s", fullResult.TokenAmountOut.Amount.String())
		}
		for _, tr := range tickRanges {
			wRes, wErr := windowedSims[tr].CloneState().(*PoolSimulator).
				CalcAmountOut(calcParams(tc.tokenIn, tc.tokenOut, tc.amountIn))
			switch {
			case wErr != nil:
				line += fmt.Sprintf(" %14s", "ERR")
			case fullErr != nil:
				line += fmt.Sprintf(" %14s", wRes.TokenAmountOut.Amount.String())
			default:
				diff := new(big.Int).Sub(fullResult.TokenAmountOut.Amount, wRes.TokenAmountOut.Amount)
				if diff.Sign() == 0 {
					line += fmt.Sprintf(" %14s", "MATCH")
				} else {
					line += fmt.Sprintf(" %14s", "diff="+diff.String())
				}
			}
		}
		t.Log(line)
	}

	t.Logf("\n=== SWAP PARITY: simulator vs CrocImpact on-chain ===")
	orderedBaseHex, orderedQuoteHex := normalizePair(valueobject.AddrZero.Hex(), usdcAddr.Hex())
	orderedBase := common.HexToAddress(orderedBaseHex)
	orderedQuote := common.HexToAddress(orderedQuoteHex)
	poolHash := EncodePoolHash(orderedBase, orderedQuote, 420)

	for _, tc := range cases {
		simRes, simErr := fullSim.CloneState().(*PoolSimulator).
			CalcAmountOut(calcParams(tc.tokenIn, tc.tokenOut, tc.amountIn))
		if simErr != nil {
			t.Logf("%-20s ERR:%s", tc.name, simErr.Error())
			continue
		}

		chainBase, chainQuote, err := callCrocImpact(h.rpcURL, orderedBase, orderedQuote, 420,
			tc.isBuy, tc.inBaseQty, tc.amountIn, h.blockBI)
		if err != nil {
			t.Logf("%-20s sim=%s RPC_ERR=%v", tc.name, simRes.TokenAmountOut.Amount, err)
			continue
		}
		chainOut := new(big.Int).Neg(chainQuote)
		if !tc.inBaseQty {
			chainOut = new(big.Int).Neg(chainBase)
		}

		// Also run simulator with live ChainBitmapView for snapshot-vs-chain diff.
		simCurve := fullState.Curve
		simSwap := &SwapDirective{
			Qty:        new(big.Int).Set(tc.amountIn),
			InBaseQty:  tc.inBaseQty,
			IsBuy:      tc.isBuy,
			LimitPrice: defaultLimitPrice(tc.isBuy),
		}
		chainBmp := &ChainBitmapView{
			Ctx: h.ctx, Client: h.client,
			DexAddr: common.HexToAddress(testLTSwapDex), PoolHash: poolHash, Block: h.blockBI,
		}
		chainAccum, err := SweepSwap(&simCurve, simSwap, &fullState.PoolParams, chainBmp)
		require.NoError(t, err)
		chainBmpOut := new(big.Int).Neg(chainAccum.QuoteFlow)
		if !tc.inBaseQty {
			chainBmpOut = new(big.Int).Neg(chainAccum.BaseFlow)
		}

		t.Logf("%-20s sim_snap=%-20s sim_chain_bmp=%-20s onchain=%-20s diff_snap=%-6s diff_bmp=%-6s",
			tc.name, simRes.TokenAmountOut.Amount, chainBmpOut, chainOut,
			new(big.Int).Sub(simRes.TokenAmountOut.Amount, chainOut),
			new(big.Int).Sub(chainBmpOut, chainOut))
	}
}

type tickLot struct {
	tick int32
	lots *big.Int
}

type poolStats struct {
	name                   string
	currentTick            int32
	minTick                int32
	maxTick                int32
	totalTicks             int
	p50, p90, p99, maxDist int32
	totalLots              *big.Int
	ticks                  []tickLot
}

func loadPoolStats(t *testing.T, h *testHarness, pairs []indexerPair) []poolStats {
	t.Helper()
	var out []poolStats
	for _, kp := range pairs {
		state, err := h.tracker.Load(h.ctx, kp.base, kp.quote, 420, h.blockBI)
		if err != nil {
			t.Logf("SKIP %s: %v", kp.name, err)
			continue
		}
		if len(state.ActiveTicks) == 0 {
			continue
		}
		current := GetTickAtSqrtRatio(state.Curve.PriceRoot)

		dists := make([]int32, len(state.ActiveTicks))
		for i, tick := range state.ActiveTicks {
			d := tick - current
			if d < 0 {
				d = -d
			}
			dists[i] = d
		}
		sort.Slice(dists, func(i, j int) bool { return dists[i] < dists[j] })

		totalLots := new(big.Int)
		ticks := make([]tickLot, 0, len(state.Levels))
		for _, lvl := range state.Levels {
			lots := new(big.Int).Add(absBI(lvl.Level.BidLots), absBI(lvl.Level.AskLots))
			if lots.Sign() == 0 {
				continue
			}
			ticks = append(ticks, tickLot{tick: lvl.Tick, lots: lots})
			totalLots.Add(totalLots, lots)
		}
		out = append(out, poolStats{
			name: kp.name, currentTick: current,
			minTick: state.ActiveTicks[0], maxTick: state.ActiveTicks[len(state.ActiveTicks)-1],
			totalTicks: len(state.ActiveTicks),
			p50:        percentile(dists, 50),
			p90:        percentile(dists, 90),
			p99:        percentile(dists, 99),
			maxDist:    dists[len(dists)-1],
			totalLots:  totalLots,
			ticks:      ticks,
		})
	}
	return out
}

func logPerPoolDist(t *testing.T, pools []poolStats) {
	t.Helper()
	t.Logf("\n=== PER-POOL TICK DISTRIBUTION (%d pools) ===", len(pools))
	t.Logf("%-20s %6s %10s %8s %8s %8s %8s", "PAIR", "TICKS", "CURRENT", "P50", "P90", "P99", "MAX")
	for _, p := range pools {
		t.Logf("%-20s %6d %10d %8d %8d %8d %8d",
			p.name, p.totalTicks, p.currentTick, p.p50, p.p90, p.p99, p.maxDist)
	}
}

type rangeStats struct {
	tickRange                            int32
	covPct                               []float64
	tickCount                            []int
	pools100, pools999, pools99, pools95 int
	worstCovPct, medianCovPct, avgCovPct float64
	medianTicks, maxTicks                int
}

func sweepRanges(pools []poolStats, candidates []int32) []rangeStats {
	stats := make([]rangeStats, len(candidates))
	for i, tr := range candidates {
		s := rangeStats{tickRange: tr, worstCovPct: 100}
		for _, p := range pools {
			minT, maxT := p.currentTick-tr, p.currentTick+tr
			covered := new(big.Int)
			tc := 0
			for _, tl := range p.ticks {
				if tl.tick >= minT && tl.tick <= maxT {
					covered.Add(covered, tl.lots)
					tc++
				}
			}
			pct := 100.0
			if p.totalLots.Sign() > 0 {
				v := new(big.Int).Mul(covered, big.NewInt(1_000_000))
				v.Div(v, p.totalLots)
				pct = float64(v.Int64()) / 10000
			}
			s.covPct = append(s.covPct, pct)
			s.tickCount = append(s.tickCount, tc)
			s.avgCovPct += pct
			if pct < s.worstCovPct {
				s.worstCovPct = pct
			}
			switch {
			case pct >= 99.999:
				s.pools100++
				fallthrough
			case pct >= 99.9:
				s.pools999++
				fallthrough
			case pct >= 99:
				s.pools99++
				fallthrough
			case pct >= 95:
				s.pools95++
			}
		}
		if n := len(pools); n > 0 {
			s.avgCovPct /= float64(n)
			sortedCov := append([]float64(nil), s.covPct...)
			sort.Float64s(sortedCov)
			s.medianCovPct = sortedCov[n/2]
			sortedTicks := append([]int(nil), s.tickCount...)
			sort.Ints(sortedTicks)
			s.medianTicks = sortedTicks[n/2]
			s.maxTicks = sortedTicks[n-1]
		}
		stats[i] = s
	}
	return stats
}

func logSweep(t *testing.T, stats []rangeStats, n int) {
	t.Helper()
	t.Logf("\n=== TICKRANGE SWEEP (liquidity-weighted, %d pools) ===", n)
	t.Logf("%-10s %8s %8s %8s %7s %7s %7s %7s %10s %10s",
		"TR", "MIN_COV", "MED_COV", "AVG_COV", ">=100%", ">=99.9%", ">=99%", ">=95%", "MED_TICKS", "MAX_TICKS")
	for _, s := range stats {
		t.Logf("%-10d %7.2f%% %7.2f%% %7.2f%% %4d/%-2d %4d/%-2d %4d/%-2d %4d/%-2d %10d %10d",
			s.tickRange, s.worstCovPct, s.medianCovPct, s.avgCovPct,
			s.pools100, n, s.pools999, n, s.pools99, n, s.pools95, n,
			s.medianTicks, s.maxTicks)
	}
}

func recommendTickRange(t *testing.T, pools []poolStats, stats []rangeStats) {
	t.Helper()
	n := len(pools)
	pick := func(poolFrac float64) (rangeStats, bool) {
		target := int(math.Ceil(float64(n) * poolFrac))
		for _, s := range stats {
			if s.pools99 >= target {
				return s, true
			}
		}
		return rangeStats{}, false
	}

	t.Logf("\n=== RECOMMENDATION (%d pools) ===", n)
	var rec rangeStats
	var haveRec bool
	for _, frac := range []float64{0.98, 0.95, 0.90, 0.80} {
		s, ok := pick(frac)
		if !ok {
			continue
		}
		t.Logf("Smallest TR where >=%.0f%% pools have >=99%% lot coverage: %d (covered=%d/%d, worst=%.2f%%, med_ticks=%d, max_ticks=%d)",
			frac*100, s.tickRange, s.pools99, n, s.worstCovPct, s.medianTicks, s.maxTicks)
		if !haveRec {
			rec, haveRec = s, true
		}
	}
	if !haveRec {
		t.Logf("No candidate covers >=80%% pools at >=99%%; extend candidates.")
		return
	}
	t.Logf("\nPools ordered by coverage at TR=%d (worst first):", rec.tickRange)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return rec.covPct[order[i]] < rec.covPct[order[j]] })
	for _, i := range order {
		t.Logf("  %-20s cov=%6.2f%% ticks=%d", pools[i].name, rec.covPct[i], rec.tickCount[i])
	}
}

func clampWindow(w TickWindow) TickWindow {
	if w.MinTick < FullTickWindow.MinTick {
		w.MinTick = FullTickWindow.MinTick
	}
	if w.MaxTick > FullTickWindow.MaxTick {
		w.MaxTick = FullTickWindow.MaxTick
	}
	return w
}

func percentile(sorted []int32, p float64) int32 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(sorted))*p/100)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func absBI(x *big.Int) *big.Int {
	if x == nil {
		return new(big.Int)
	}
	if x.Sign() < 0 {
		return new(big.Int).Neg(x)
	}
	return new(big.Int).Set(x)
}

// --- indexer / on-chain helpers --------------------------------------------

type indexerPair struct {
	name  string
	base  common.Address
	quote common.Address
}

func fetchIndexerPairs(t *testing.T, poolIdx int) []indexerPair {
	t.Helper()
	type indexerPool struct {
		Base    string `json:"base"`
		Quote   string `json:"quote"`
		PoolIdx int    `json:"poolIdx"`
	}
	var body struct {
		Data []indexerPool `json:"data"`
	}
	resp, err := http.Get("https://ambindexer.net/gcgo/pool_list?chainId=0x1") //nolint:gosec
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	pairs := make([]indexerPair, 0, len(body.Data))
	for _, p := range body.Data {
		if p.PoolIdx != poolIdx {
			continue
		}
		pairs = append(pairs, indexerPair{
			name:  fmt.Sprintf("%s/%s", p.Base[:8], p.Quote[:8]),
			base:  common.HexToAddress(p.Base),
			quote: common.HexToAddress(p.Quote),
		})
	}
	return pairs
}

func buildSimulator(t *testing.T, state *TrackerExtra, wethAddr common.Address) *PoolSimulator {
	t.Helper()
	staticExtra, err := json.Marshal(StaticExtra{
		NativeToken: wethAddr.Hex(), PoolIdx: 420, SwapDex: testLTSwapDex,
		Base: state.Base.Hex(), Quote: state.Quote.Hex(),
	})
	require.NoError(t, err)
	extra, err := json.Marshal(Extra{State: state})
	require.NoError(t, err)

	token0 := wethAddr.Hex()
	if state.Base != valueobject.AddrZero {
		token0 = state.Base.Hex()
	}
	token1 := state.Quote.Hex()
	if state.Quote == valueobject.AddrZero {
		token1 = wethAddr.Hex()
	}
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     state.PoolHash.Hex(),
		Exchange:    DexType,
		Type:        DexType,
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		Reserves: []string{"1000000000000000000000", "1000000000000"},
	})
	require.NoError(t, err)
	return sim
}

func calcParams(tokenIn, tokenOut string, amountIn *big.Int) pool.CalcAmountOutParams {
	return pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amountIn)},
		TokenOut:      tokenOut,
	}
}

const crocImpactABI = `[{
	"inputs":[
		{"name":"base","type":"address"},
		{"name":"quote","type":"address"},
		{"name":"poolIdx","type":"uint256"},
		{"name":"isBuy","type":"bool"},
		{"name":"inBaseQty","type":"bool"},
		{"name":"qty","type":"uint128"},
		{"name":"poolTip","type":"uint16"},
		{"name":"limitPrice","type":"uint128"}
	],
	"name":"calcImpact",
	"outputs":[
		{"name":"baseFlow","type":"int128"},
		{"name":"quoteFlow","type":"int128"},
		{"name":"finalPrice","type":"uint128"}
	],
	"stateMutability":"view","type":"function"
}]`

func callCrocImpact(
	rpcURL string,
	base, quote common.Address,
	poolIdx uint64,
	isBuy, inBaseQty bool,
	qty, blockNum *big.Int,
) (baseFlow, quoteFlow *big.Int, err error) {
	parsed, _ := abi.JSON(strings.NewReader(crocImpactABI))
	limitPrice := new(big.Int).Set(MinSqrtRatio)
	if isBuy {
		limitPrice = new(big.Int).Sub(MaxSqrtRatio, big.NewInt(1))
	}
	data, err := parsed.Pack("calcImpact",
		base, quote, new(big.Int).SetUint64(poolIdx),
		isBuy, inBaseQty, qty, uint16(0), limitPrice)
	if err != nil {
		return nil, nil, fmt.Errorf("pack: %w", err)
	}

	raw, err := jsonRPCCall(rpcURL, "eth_call", []any{
		map[string]string{"to": crocImpactAddr, "data": "0x" + common.Bytes2Hex(data)},
		fmt.Sprintf("0x%x", blockNum.Uint64()),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("eth_call: %w", err)
	}
	var hex string
	if err := json.Unmarshal(raw, &hex); err != nil {
		return nil, nil, fmt.Errorf("unmarshal: %w", err)
	}
	out, err := parsed.Unpack("calcImpact", common.FromHex(hex))
	if err != nil {
		return nil, nil, fmt.Errorf("unpack: %w", err)
	}
	return out[0].(*big.Int), out[1].(*big.Int), nil
}

func jsonRPCCall(rpcURL, method string, params []any) (json.RawMessage, error) {
	data, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
	})
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(data)) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", res.Error.Code, res.Error.Message)
	}
	return res.Result, nil
}
