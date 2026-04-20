package ambient

import (
	"bytes"
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

// TestTickRangeAnalysis loads all known pairs and shows tick distribution
// relative to current price to help choose a good TickRange.
func TestTickRangeAnalysis(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	rpcEndpoint := os.Getenv("ETHEREUM_RPC_URL")
	if rpcEndpoint == "" {
		rpcEndpoint = testLTRPCURL
	}

	type knownPair struct {
		name  string
		base  common.Address
		quote common.Address
	}
	knownPairs := []knownPair{
		{"ETH/USDC", valueobject.AddrZero, common.HexToAddress(testLTUSDC)},
		{"ETH/USDT", valueobject.AddrZero, common.HexToAddress(testLTUSDT)},
		{"ETH/WBTC", valueobject.AddrZero, common.HexToAddress(testLTWBTC)},
		{"USDC/USDT", common.HexToAddress(testLTUSDC), common.HexToAddress(testLTUSDT)},
		{"WBTC/USDC", common.HexToAddress(testLTWBTC), common.HexToAddress(testLTUSDC)},
		{"WBTC/USDT", common.HexToAddress(testLTWBTC), common.HexToAddress(testLTUSDT)},
	}

	ctx := t.Context()
	client, err := ethclient.DialContext(ctx, rpcEndpoint)
	require.NoError(t, err)
	defer client.Close()

	tracker := NewStateTracker(client, testLTSwapDex)

	blockNum, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	blockBI := new(big.Int).SetUint64(blockNum)
	t.Logf("block: %d", blockNum)

	type pairResult struct {
		pair        string
		totalTicks  int
		currentTick int32
		minTick     int32
		maxTick     int32
		p50dist     int32
		p90dist     int32
		p99dist     int32
		maxDist     int32
	}

	var results []pairResult

	for _, kp := range knownPairs {
		state, err := tracker.Load(ctx, kp.base, kp.quote, 420, blockBI)
		if err != nil {
			t.Logf("SKIP %s: %v", kp.name, err)
			continue
		}
		if len(state.ActiveTicks) == 0 {
			t.Logf("SKIP %s: 0 active ticks", kp.name)
			continue
		}

		currentTick := GetTickAtSqrtRatio(state.Curve.PriceRoot)

		distances := make([]int32, len(state.ActiveTicks))
		for i, tick := range state.ActiveTicks {
			d := tick - currentTick
			if d < 0 {
				d = -d
			}
			distances[i] = d
		}
		sort.Slice(distances, func(i, j int) bool { return distances[i] < distances[j] })

		percentile := func(p float64) int32 {
			idx := int(math.Ceil(float64(len(distances))*p/100)) - 1
			if idx < 0 {
				idx = 0
			}
			if idx >= len(distances) {
				idx = len(distances) - 1
			}
			return distances[idx]
		}

		r := pairResult{
			pair:        kp.name,
			totalTicks:  len(state.ActiveTicks),
			currentTick: currentTick,
			minTick:     state.ActiveTicks[0],
			maxTick:     state.ActiveTicks[len(state.ActiveTicks)-1],
			p50dist:     percentile(50),
			p90dist:     percentile(90),
			p99dist:     percentile(99),
			maxDist:     distances[len(distances)-1],
		}
		results = append(results, r)

		t.Logf("%-12s ticks=%-5d current=%-8d range=[%d, %d] p50=%-6d p90=%-6d p99=%-6d max=%-6d",
			r.pair, r.totalTicks, r.currentTick,
			r.minTick, r.maxTick,
			r.p50dist, r.p90dist, r.p99dist, r.maxDist)
	}

	t.Logf("\n=== RECOMMENDATION ===")
	t.Logf("%-12s %6s %6s %6s %6s %9s", "PAIR", "TICKS", "P90", "P99", "MAX", "SUGGESTED")
	for _, r := range results {
		suggested := ((int32(float64(r.p99dist)*1.2) / 10000) + 1) * 10000
		t.Logf("%-12s %6d %6d %6d %6d %9d",
			r.pair, r.totalTicks, r.p90dist, r.p99dist, r.maxDist, suggested)
	}
}

// TestTickRangeComparison loads ETH/USDC at different TickRange values and
// shows how many ticks are captured.
func TestTickRangeComparison(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	rpcEndpoint := os.Getenv("ETHEREUM_RPC_URL")
	if rpcEndpoint == "" {
		rpcEndpoint = testLTRPCURL
	}

	ctx := t.Context()
	client, err := ethclient.DialContext(ctx, rpcEndpoint)
	require.NoError(t, err)
	defer client.Close()

	tracker := NewStateTracker(client, testLTSwapDex)

	blockNum, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	blockBI := new(big.Int).SetUint64(blockNum)

	base := valueobject.AddrZero
	quote := common.HexToAddress(testLTUSDC)

	fullState, err := tracker.Load(ctx, base, quote, 420, blockBI)
	require.NoError(t, err)
	t.Logf("full load: %d ticks", len(fullState.ActiveTicks))

	currentTick := GetTickAtSqrtRatio(fullState.Curve.PriceRoot)

	for _, tr := range []int32{10000, 20000, 50000, 100000, 200000} {
		window := TickWindow{
			MinTick: currentTick - tr,
			MaxTick: currentTick + tr,
		}
		if window.MinTick < FullTickWindow.MinTick {
			window.MinTick = FullTickWindow.MinTick
		}
		if window.MaxTick > FullTickWindow.MaxTick {
			window.MaxTick = FullTickWindow.MaxTick
		}

		windowed, err := tracker.LoadWindow(ctx, base, quote, 420, blockBI, window)
		require.NoError(t, err)

		pct := float64(len(windowed.ActiveTicks)) / float64(len(fullState.ActiveTicks)) * 100
		t.Logf("TickRange=%-7d → %3d / %3d ticks (%.1f%%) window=[%d, %d]",
			tr, len(windowed.ActiveTicks), len(fullState.ActiveTicks), pct,
			window.MinTick, window.MaxTick)
	}
}

// TestTickRangeSwapParity compares simulator CalcAmountOut across different
// TickRange values against:
// 1. Full-range simulator (TickRange=0) as baseline
// 2. On-chain CrocImpact.calcImpact() for ground truth
//
// This verifies that windowed loading doesn't change swap results for
// reasonable swap sizes.
func TestTickRangeSwapParity(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	rpcEndpoint := os.Getenv("ETHEREUM_RPC_URL")
	if rpcEndpoint == "" {
		rpcEndpoint = testLTRPCURL
	}

	ctx := t.Context()
	client, err := ethclient.DialContext(ctx, rpcEndpoint)
	require.NoError(t, err)
	defer client.Close()

	tracker := NewStateTracker(client, testLTSwapDex)

	blockNum, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	blockBI := new(big.Int).SetUint64(blockNum)
	t.Logf("block: %d", blockNum)

	base := valueobject.AddrZero
	quote := common.HexToAddress(testLTUSDC)

	// Load full state.
	fullState, err := tracker.Load(ctx, base, quote, 420, blockBI)
	require.NoError(t, err)

	currentTick := GetTickAtSqrtRatio(fullState.Curve.PriceRoot)
	t.Logf("ETH/USDC: %d ticks, currentTick=%d, price=%s",
		len(fullState.ActiveTicks), currentTick, fullState.Curve.PriceRoot)

	// Load windowed states.
	tickRanges := []int32{10000, 20000, 50000}
	windowedStates := make(map[int32]*TrackerExtra)
	for _, tr := range tickRanges {
		window := TickWindow{
			MinTick: currentTick - tr,
			MaxTick: currentTick + tr,
		}
		if window.MinTick < FullTickWindow.MinTick {
			window.MinTick = FullTickWindow.MinTick
		}
		if window.MaxTick > FullTickWindow.MaxTick {
			window.MaxTick = FullTickWindow.MaxTick
		}
		ws, err := tracker.LoadWindow(ctx, base, quote, 420, blockBI, window)
		require.NoError(t, err)
		windowedStates[tr] = ws
		t.Logf("TickRange=%d: %d ticks", tr, len(ws.ActiveTicks))
	}

	// Swap test cases: various amounts, both directions.
	// ETH amounts in wei, USDC amounts in 6-decimal units.
	type swapCase struct {
		name      string
		tokenIn   string
		tokenOut  string
		amountIn  *big.Int
		isBuy     bool
		inBaseQty bool
	}

	wethAddr := common.HexToAddress(testLTNativeTokenAddress)
	usdcAddr := common.HexToAddress(testLTUSDC)

	wei18 := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	wei6 := big.NewInt(1_000_000)

	cases := []swapCase{
		{"0.01 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Div(wei18, big.NewInt(100)), true, true},
		{"0.1 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Div(wei18, big.NewInt(10)), true, true},
		{"1 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Set(wei18), true, true},
		{"10 ETH→USDC", wethAddr.Hex(), usdcAddr.Hex(), new(big.Int).Mul(big.NewInt(10), wei18), true, true},
		{"100 USDC→ETH", usdcAddr.Hex(), wethAddr.Hex(), new(big.Int).Mul(big.NewInt(100), wei6), false, false},
		{"1000 USDC→ETH", usdcAddr.Hex(), wethAddr.Hex(), new(big.Int).Mul(big.NewInt(1000), wei6), false, false},
		{"10000 USDC→ETH", usdcAddr.Hex(), wethAddr.Hex(), new(big.Int).Mul(big.NewInt(10000), wei6), false, false},
	}

	// Build simulator from full state.
	fullSim := buildSimulator(t, fullState, wethAddr)

	// Build simulators from windowed states.
	windowedSims := make(map[int32]*PoolSimulator)
	for tr, ws := range windowedStates {
		windowedSims[tr] = buildSimulator(t, ws, wethAddr)
	}

	// Header.
	t.Logf("\n=== SWAP PARITY: simulator(full) vs simulator(windowed) ===")
	header := fmt.Sprintf("%-20s %18s", "CASE", "FULL_OUT")
	for _, tr := range tickRanges {
		header += fmt.Sprintf(" %14s", fmt.Sprintf("TR=%d", tr))
	}
	t.Log(header)

	for _, tc := range cases {
		// Full range result.
		fullResult, fullErr := fullSim.CloneState().(*PoolSimulator).CalcAmountOut(calcParams(tc.tokenIn, tc.tokenOut, tc.amountIn))

		line := fmt.Sprintf("%-20s", tc.name)
		if fullErr != nil {
			line += fmt.Sprintf(" %18s", "ERR:"+fullErr.Error())
		} else {
			line += fmt.Sprintf(" %18s", fullResult.TokenAmountOut.Amount.String())
		}

		for _, tr := range tickRanges {
			sim := windowedSims[tr].CloneState().(*PoolSimulator)
			wResult, wErr := sim.CalcAmountOut(calcParams(tc.tokenIn, tc.tokenOut, tc.amountIn))

			if wErr != nil {
				line += fmt.Sprintf(" %14s", "ERR")
			} else if fullErr != nil {
				line += fmt.Sprintf(" %14s", wResult.TokenAmountOut.Amount.String())
			} else {
				diff := new(big.Int).Sub(fullResult.TokenAmountOut.Amount, wResult.TokenAmountOut.Amount)
				if diff.Sign() == 0 {
					line += fmt.Sprintf(" %14s", "MATCH")
				} else {
					line += fmt.Sprintf(" %14s", "diff="+diff.String())
				}
			}
		}
		t.Log(line)
	}

	// On-chain comparison via CrocImpact.
	t.Logf("\n=== SWAP PARITY: simulator(full) vs CrocImpact on-chain ===")
	t.Logf("%-20s %18s %18s %10s", "CASE", "SIM_OUT", "CHAIN_OUT", "DIFF")

	orderedBaseHex, orderedQuoteHex := normalizePair(valueobject.AddrZero.Hex(), usdcAddr.Hex())
	orderedBase := common.HexToAddress(orderedBaseHex)
	orderedQuote := common.HexToAddress(orderedQuoteHex)
	poolHash := EncodePoolHash(orderedBase, orderedQuote, 420)

	for _, tc := range cases {
		fullClone := fullSim.CloneState().(*PoolSimulator)
		simResult, simErr := fullClone.CalcAmountOut(calcParams(tc.tokenIn, tc.tokenOut, tc.amountIn))
		if simErr != nil {
			t.Logf("%-20s %18s %18s %10s", tc.name, "ERR:"+simErr.Error(), "-", "-")
			continue
		}

		// Call CrocImpact on-chain.
		chainBase, chainQuote, err := callCrocImpact(
			rpcEndpoint, orderedBase, orderedQuote, 420,
			tc.isBuy, tc.inBaseQty, tc.amountIn, blockBI,
		)
		if err != nil {
			t.Logf("%-20s %18s %18s %10s", tc.name, simResult.TokenAmountOut.Amount.String(), "RPC_ERR", err.Error())
			continue
		}

		// Derive on-chain output.
		var chainOut *big.Int
		if tc.inBaseQty {
			chainOut = new(big.Int).Neg(chainQuote)
		} else {
			chainOut = new(big.Int).Neg(chainBase)
		}

		simOut := simResult.TokenAmountOut.Amount

		// Run simulator with ChainBitmapView for ground truth comparison.
		simCurve := fullState.Curve
		simSwap := &SwapDirective{
			Qty:        new(big.Int).Set(tc.amountIn),
			InBaseQty:  tc.inBaseQty,
			IsBuy:      tc.isBuy,
			LimitPrice: defaultLimitPrice(tc.isBuy),
		}
		chainBmp := &ChainBitmapView{
			Ctx:      ctx,
			Client:   client,
			DexAddr:  common.HexToAddress(testLTSwapDex),
			PoolHash: poolHash,
			Block:    blockBI,
		}
		chainAccum, err := SweepSwap(&simCurve, simSwap, &fullState.PoolParams, chainBmp)
		if err != nil {
			t.Fatalf("SweepSwap: %v", err)
		}
		var chainBmpOut *big.Int
		if tc.inBaseQty {
			chainBmpOut = new(big.Int).Neg(chainAccum.QuoteFlow)
		} else {
			chainBmpOut = new(big.Int).Neg(chainAccum.BaseFlow)
		}

		diff := new(big.Int).Sub(simOut, chainOut)
		diffBmp := new(big.Int).Sub(chainBmpOut, chainOut)

		t.Logf("%-20s sim_snap=%-18s sim_chain_bmp=%-18s onchain=%-18s diff_snap=%-6s diff_bmp=%-6s",
			tc.name,
			simOut.String(),
			chainBmpOut.String(),
			chainOut.String(),
			diff.String(),
			diffBmp.String(),
		)
	}
}

// TestTickRangeReserveCoverage fetches all pools from ambindexer API and shows
// what percentage of concentrated liquidity each TickRange covers.
func TestTickRangeReserveCoverage(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	rpcEndpoint := os.Getenv("ETHEREUM_RPC_URL")
	if rpcEndpoint == "" {
		rpcEndpoint = testLTRPCURL
	}

	ctx := t.Context()
	client, err := ethclient.DialContext(ctx, rpcEndpoint)
	require.NoError(t, err)
	defer client.Close()

	tracker := NewStateTracker(client, testLTSwapDex)

	blockNum, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	blockBI := new(big.Int).SetUint64(blockNum)
	t.Logf("block: %d", blockNum)

	// Fetch all pools from ambindexer API.
	type indexerPool struct {
		Base    string `json:"base"`
		Quote   string `json:"quote"`
		PoolIdx int    `json:"poolIdx"`
	}
	type indexerResp struct {
		Data []indexerPool `json:"data"`
	}

	resp, err := http.Get("https://ambindexer.net/gcgo/pool_list?chainId=0x1") //nolint:gosec
	require.NoError(t, err)
	defer resp.Body.Close()

	var indexer indexerResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&indexer))
	t.Logf("ambindexer returned %d pools", len(indexer.Data))

	type knownPair struct {
		name  string
		base  common.Address
		quote common.Address
	}
	var pairs []knownPair
	for _, p := range indexer.Data {
		if p.PoolIdx != 420 {
			continue
		}
		base := common.HexToAddress(p.Base)
		quote := common.HexToAddress(p.Quote)
		name := fmt.Sprintf("%s/%s", p.Base[:8], p.Quote[:8])
		pairs = append(pairs, knownPair{name: name, base: base, quote: quote})
	}

	tickRanges := []int32{5000, 10000, 20000, 50000, 100000}

	// Header.
	header := fmt.Sprintf("%-12s %6s %14s", "PAIR", "TICKS", "TOTAL_LOTS")
	for _, tr := range tickRanges {
		header += fmt.Sprintf(" %12s", fmt.Sprintf("TR=%d", tr))
	}
	t.Log(header)

	for _, kp := range pairs {
		state, err := tracker.Load(ctx, kp.base, kp.quote, 420, blockBI)
		if err != nil {
			t.Logf("SKIP %s: %v", kp.name, err)
			continue
		}
		if len(state.ActiveTicks) == 0 {
			t.Logf("SKIP %s: 0 ticks", kp.name)
			continue
		}

		currentTick := GetTickAtSqrtRatio(state.Curve.PriceRoot)

		// Build tick → lots map.
		type tickLots struct {
			tick int32
			lots *big.Int // bidLots + askLots
		}
		var allTicks []tickLots
		totalLots := new(big.Int)
		for _, level := range state.Levels {
			lots := new(big.Int).Add(
				absBI(level.Level.BidLots),
				absBI(level.Level.AskLots),
			)
			if lots.Sign() > 0 {
				allTicks = append(allTicks, tickLots{tick: level.Tick, lots: lots})
				totalLots.Add(totalLots, lots)
			}
		}

		line := fmt.Sprintf("%-12s %6d %14s", kp.name, len(state.ActiveTicks), totalLots.String())

		for _, tr := range tickRanges {
			minT := currentTick - tr
			maxT := currentTick + tr

			coveredLots := new(big.Int)
			coveredTicks := 0
			for _, tl := range allTicks {
				if tl.tick >= minT && tl.tick <= maxT {
					coveredLots.Add(coveredLots, tl.lots)
					coveredTicks++
				}
			}

			var pct float64
			if totalLots.Sign() > 0 {
				pctBI := new(big.Int).Mul(coveredLots, big.NewInt(10000))
				pctBI.Div(pctBI, totalLots)
				pct = float64(pctBI.Int64()) / 100
			}
			line += fmt.Sprintf(" %5d/%5d=%5.1f%%", coveredTicks, len(allTicks), pct)
		}
		t.Log(line)

		// Detailed: show liquidity distribution in bands.
		t.Logf("  Liquidity distribution around currentTick=%d:", currentTick)
		bands := []struct {
			label string
			lo    int32
			hi    int32
		}{
			{"±1000", currentTick - 1000, currentTick + 1000},
			{"±5000", currentTick - 5000, currentTick + 5000},
			{"±10000", currentTick - 10000, currentTick + 10000},
			{"±20000", currentTick - 20000, currentTick + 20000},
			{"outer", FullTickWindow.MinTick, FullTickWindow.MaxTick},
		}
		for _, band := range bands {
			bandLots := new(big.Int)
			bandTicks := 0
			for _, tl := range allTicks {
				inBand := tl.tick >= band.lo && tl.tick <= band.hi
				if band.label == "outer" {
					inBand = tl.tick < (currentTick-20000) || tl.tick > (currentTick+20000)
				}
				if inBand {
					bandLots.Add(bandLots, tl.lots)
					bandTicks++
				}
			}
			var pct float64
			if totalLots.Sign() > 0 {
				pctBI := new(big.Int).Mul(bandLots, big.NewInt(10000))
				pctBI.Div(pctBI, totalLots)
				pct = float64(pctBI.Int64()) / 100
			}
			t.Logf("    %8s: %4d ticks, lots=%s (%.1f%%)", band.label, bandTicks, bandLots, pct)
		}
	}
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

func buildSimulator(t *testing.T, state *TrackerExtra, wethAddr common.Address) *PoolSimulator {
	t.Helper()

	staticExtra, err := json.Marshal(StaticExtra{
		NativeToken: wethAddr.Hex(),
		PoolIdx:     420,
		SwapDex:     testLTSwapDex,
		Base:        state.Base.Hex(),
		Quote:       state.Quote.Hex(),
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

	sim, err := NewPoolSimulator(makeEntityPool(
		state.PoolHash.Hex(),
		token0,
		token1,
		string(staticExtra),
		string(extra),
	))
	require.NoError(t, err)
	return sim
}

func makeEntityPool(address, token0, token1, staticExtraStr, extraStr string) entity.Pool {
	return entity.Pool{
		Address:     address,
		Exchange:    DexType,
		Type:        DexType,
		StaticExtra: staticExtraStr,
		Extra:       extraStr,
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		Reserves: []string{"1000000000000000000000", "1000000000000"},
	}
}

func calcParams(tokenIn, tokenOut string, amountIn *big.Int) pool.CalcAmountOutParams {
	return pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amountIn)},
		TokenOut:      tokenOut,
	}
}

func callCrocImpact(
	rpcURL string,
	base, quote common.Address,
	poolIdx uint64,
	isBuy, inBaseQty bool,
	qty, blockNum *big.Int,
) (baseFlow, quoteFlow *big.Int, err error) {
	const impactABIJSON = `[{
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
		"stateMutability":"view",
		"type":"function"
	}]`

	parsed, _ := abi.JSON(strings.NewReader(impactABIJSON))

	var limitPrice *big.Int
	if isBuy {
		limitPrice = new(big.Int).Sub(MaxSqrtRatio, big.NewInt(1))
	} else {
		limitPrice = new(big.Int).Set(MinSqrtRatio)
	}

	data, err := parsed.Pack("calcImpact",
		base, quote, new(big.Int).SetUint64(poolIdx),
		isBuy, inBaseQty, qty,
		uint16(0), limitPrice,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("pack: %w", err)
	}

	blockHex := fmt.Sprintf("0x%x", blockNum.Uint64())
	result, err := jsonRPCCall(rpcURL, "eth_call", []any{
		map[string]string{
			"to":   crocImpactAddr,
			"data": "0x" + common.Bytes2Hex(data),
		},
		blockHex,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("eth_call: %w", err)
	}

	var hex string
	if err := json.Unmarshal(result, &hex); err != nil {
		return nil, nil, fmt.Errorf("unmarshal: %w", err)
	}

	out, err := parsed.Unpack("calcImpact", common.FromHex(hex))
	if err != nil {
		return nil, nil, fmt.Errorf("unpack: %w", err)
	}

	return out[0].(*big.Int), out[1].(*big.Int), nil
}

func jsonRPCCall(rpcURL, method string, params []any) (json.RawMessage, error) {
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(body)
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
