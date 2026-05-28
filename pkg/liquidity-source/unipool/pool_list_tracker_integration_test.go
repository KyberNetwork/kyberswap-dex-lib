package unipool

import (
	"bytes"
	"context"
	_ "embed"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

//go:embed abis/UniPoolQuoter.json
var quoterABIJsonForTest []byte

// Live RPC integration tests against UniPool on Arbitrum.
//
// These tests follow the repo convention (cf. ekubo, nabla): they are committed
// to the package but skipped when the CI env var is set (test.SkipCI), so they
// run only on developer machines. To target a private RPC (Alchemy etc.), set:
//
//	export ARBITRUM_RPC_URL="https://arb-mainnet.g.alchemy.com/v2/<KEY>"
//
// Default falls back to KyberSwap's public Arbitrum endpoint.

const (
	arbitrumDefaultRPC     = "https://arbitrum-rpc.kyberswap.com"
	arbitrumMulticall3     = "0xcA11bde05977b3631167028862bE2a173976CA11"
	uniPoolArbFactoryAddr  = "0xa88216E6Cf409a25c719234C4817628Ae406b6A7"
	uniPoolArbQuoterAddr   = "0xc264944e9e7073f8f98fef7338cda973914fca44"
	uniPoolArbWETHUSDTPair = "0x6Ce2b09bB578137130E17A0476a6adcf0ac7b0da"
	uniPoolArbEVUSDTPair   = "0xfa896ef9659ea0dcf42c751e2b1f78f626fe8f56"
	arbWETHAddr            = "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"
	arbUSDTAddr            = "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"
)

type PoolListTrackerIntegrationSuite struct {
	suite.Suite

	client  *ethrpc.Client
	lister  *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *PoolListTrackerIntegrationSuite) SetupSuite() {
	rpcURL := os.Getenv("ARBITRUM_RPC_URL")
	if rpcURL == "" {
		rpcURL = arbitrumDefaultRPC
	}
	rpcClient := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress(arbitrumMulticall3))
	ts.client = rpcClient

	cfg := &Config{
		DexID:          DexType,
		FactoryAddress: uniPoolArbFactoryAddr,
		NewPoolLimit:   50,
	}
	ts.lister = NewPoolsListUpdater(cfg, rpcClient)

	tracker, err := NewPoolTracker(cfg, rpcClient)
	ts.Require().NoError(err)
	ts.tracker = tracker
}

// TestListPools_ReturnsNonEmptyBatch verifies the factory pagination path.
func (ts *PoolListTrackerIntegrationSuite) TestListPools_ReturnsNonEmptyBatch() {
	pools, meta, err := ts.lister.GetNewPools(context.Background(), nil)
	ts.Require().NoError(err)
	ts.Require().NotEmpty(pools, "factory should expose at least one pair")
	ts.Require().NotEmpty(meta, "metadata should advance offset")

	for i, p := range pools {
		ts.Require().NotEmptyf(p.Address, "pool[%d].Address empty", i)
		ts.Require().Equalf(2, len(p.Tokens), "pool[%d] must have 2 tokens", i)
		ts.Require().NotEmptyf(p.Tokens[0].Address, "pool[%d].Tokens[0] empty", i)
		ts.Require().NotEmptyf(p.Tokens[1].Address, "pool[%d].Tokens[1] empty", i)
		ts.Require().Equalf(DexType, p.Type, "pool[%d].Type", i)
		ts.Require().Equalf(DexType, p.Exchange, "pool[%d].Exchange", i)

		// StaticExtra should embed our factory address.
		ts.Require().Containsf(strings.ToLower(p.StaticExtra),
			strings.ToLower(uniPoolArbFactoryAddr),
			"pool[%d].StaticExtra missing factory address", i)
	}
}

// TestListPools_PaginationAdvances verifies metadata-driven incremental paging.
// On a small factory we may exhaust everything in the first call; in that case
// the second call returns 0 pools and the offset must stay (not regress).
func (ts *PoolListTrackerIntegrationSuite) TestListPools_PaginationAdvances() {
	pools1, meta1, err := ts.lister.GetNewPools(context.Background(), nil)
	ts.Require().NoError(err)
	ts.Require().NotEmpty(pools1)

	pools2, meta2, err := ts.lister.GetNewPools(context.Background(), meta1)
	ts.Require().NoError(err)

	if len(pools2) > 0 {
		// More pages exist: first address must change AND metadata must advance.
		ts.Require().NotEqual(pools1[0].Address, pools2[0].Address,
			"pagination should advance, got the same first pool twice")
		ts.Require().NotEqualf(string(meta1), string(meta2),
			"metadata must advance when new pools are returned")
	} else {
		// Factory exhausted: metadata must NOT regress (offset stays put).
		var m1, m2 PoolsListUpdaterMetadata
		ts.Require().NoError(json.Unmarshal(meta1, &m1))
		ts.Require().NoError(json.Unmarshal(meta2, &m2))
		ts.Require().Equalf(m1.Offset, m2.Offset,
			"empty second batch must leave offset unchanged, got %d -> %d",
			m1.Offset, m2.Offset)
	}
}

// TestTrackWETHUSDT verifies the tracker populates Extra on a live pair with
// reserves > 0. WETH/USDT on Arbitrum should always have liquidity.
func (ts *PoolListTrackerIntegrationSuite) TestTrackWETHUSDT() {
	updated, err := ts.tracker.GetNewPoolState(
		context.Background(),
		entity.Pool{
			Address:  uniPoolArbWETHUSDTPair,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: arbWETHAddr, Swappable: true},
				{Address: arbUSDTAddr, Swappable: true},
			},
		},
		pool.GetNewPoolStateParams{},
	)
	ts.Require().NoError(err)

	var extra Extra
	ts.Require().NoError(json.Unmarshal([]byte(updated.Extra), &extra))

	ts.assertNonZeroReserves(extra, "WETH/USDT")
	ts.assertValidExtra(extra, "WETH/USDT")

	// pool.Reserves slice should also reflect the on-chain reserves.
	ts.Require().Equal(extra.Reserve0.String(), string(updated.Reserves[0]))
	ts.Require().Equal(extra.Reserve1.String(), string(updated.Reserves[1]))

	// The simulator should accept this snapshot and quote a small swap.
	sim, err := NewPoolSimulator(updated)
	ts.Require().NoError(err)

	// 0.001 WETH (1e15) in -> some USDT out.
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: arbWETHAddr, Amount: big.NewInt(1e15)},
		TokenOut:      arbUSDTAddr,
	})
	ts.Require().NoError(err)
	ts.Require().NotNil(res.TokenAmountOut)
	ts.Require().Equal(1, res.TokenAmountOut.Amount.Sign(), "amountOut must be > 0")
	ts.T().Logf("[WETH/USDT] quote: 1e15 WETH -> %s USDT (gas=%d)",
		res.TokenAmountOut.Amount.String(), res.Gas)
}

// TestTrackEVUSDT verifies the tracker on the second known pair. EV/USDT may
// have lower or zero liquidity depending on the moment — we only assert
// structural validity, not reserve values.
func (ts *PoolListTrackerIntegrationSuite) TestTrackEVUSDT() {
	updated, err := ts.tracker.GetNewPoolState(
		context.Background(),
		entity.Pool{
			Address:  uniPoolArbEVUSDTPair,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
		},
		pool.GetNewPoolStateParams{},
	)
	ts.Require().NoError(err)

	var extra Extra
	ts.Require().NoError(json.Unmarshal([]byte(updated.Extra), &extra))

	ts.assertValidExtra(extra, "EV/USDT")
	ts.T().Logf("[EV/USDT] reserve0=%s reserve1=%s priceDecay=%d lastUpdate=%d fees=(%d+%d)",
		extra.Reserve0, extra.Reserve1,
		extra.PriceDecay, extra.LastUpdateTimestamp,
		extra.FeeLpBps, extra.FeePoolBps)
}

// TestEndToEnd_QuoteFromListedPool exercises the full lifecycle: discover via
// the lister, refresh via the tracker, build a simulator, run a quote.
func (ts *PoolListTrackerIntegrationSuite) TestEndToEnd_QuoteFromListedPool() {
	pools, _, err := ts.lister.GetNewPools(context.Background(), nil)
	ts.Require().NoError(err)

	var found *entity.Pool
	for i := range pools {
		if strings.EqualFold(pools[i].Address, uniPoolArbWETHUSDTPair) {
			found = &pools[i]
			break
		}
	}
	if found == nil {
		// WETH/USDT may be past the first page; fetch more pages until we hit it.
		meta, _ := json.Marshal(PoolsListUpdaterMetadata{Offset: ts.lister.config.NewPoolLimit})
		for round := 0; round < 5 && found == nil; round++ {
			next, m, err := ts.lister.GetNewPools(context.Background(), meta)
			ts.Require().NoError(err)
			if len(next) == 0 {
				break
			}
			for i := range next {
				if strings.EqualFold(next[i].Address, uniPoolArbWETHUSDTPair) {
					found = &next[i]
					break
				}
			}
			meta = m
		}
	}
	ts.Require().NotNil(found, "WETH/USDT pair not exposed by factory pagination")

	tracked, err := ts.tracker.GetNewPoolState(context.Background(), *found,
		pool.GetNewPoolStateParams{})
	ts.Require().NoError(err)

	sim, err := NewPoolSimulator(tracked)
	ts.Require().NoError(err)

	// Quote both directions.
	for _, dir := range []struct {
		in, out string
		amount  *big.Int
	}{
		{found.Tokens[0].Address, found.Tokens[1].Address, big.NewInt(1e15)},
		{found.Tokens[1].Address, found.Tokens[0].Address, big.NewInt(1e9)},
	} {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: dir.in, Amount: dir.amount},
			TokenOut:      dir.out,
		})
		ts.Require().NoErrorf(err, "quote %s->%s failed", dir.in[:8], dir.out[:8])
		ts.Require().Equal(1, res.TokenAmountOut.Amount.Sign())
		ts.T().Logf("[E2E] %s -> %s : %s -> %s",
			dir.in[:8], dir.out[:8], dir.amount, res.TokenAmountOut.Amount)
	}
}

// ---- helpers --------------------------------------------------------------

func (ts *PoolListTrackerIntegrationSuite) assertNonZeroReserves(e Extra, label string) {
	ts.Require().Equalf(1, e.Reserve0.Sign(), "%s: reserve0 should be > 0", label)
	ts.Require().Equalf(1, e.Reserve1.Sign(), "%s: reserve1 should be > 0", label)
}

func (ts *PoolListTrackerIntegrationSuite) assertValidExtra(e Extra, label string) {
	ts.Require().NotNilf(e.Reserve0, "%s: Reserve0 nil", label)
	ts.Require().NotNilf(e.Reserve1, "%s: Reserve1 nil", label)
	ts.Require().NotNilf(e.VirtualReserve0In, "%s: VR0In nil", label)
	ts.Require().NotNilf(e.VirtualReserve0Out, "%s: VR0Out nil", label)
	ts.Require().NotNilf(e.VirtualReserve1In, "%s: VR1In nil", label)
	ts.Require().NotNilf(e.VirtualReserve1Out, "%s: VR1Out nil", label)
	ts.Require().NotNilf(e.TotalBorrowed0, "%s: TotalBorrowed0 nil", label)
	ts.Require().NotNilf(e.TotalBorrowed1, "%s: TotalBorrowed1 nil", label)

	// Fees should be in a sane range (sum strictly < BPS_DIVISOR).
	totalFee := uint32(e.FeeLpBps) + uint32(e.FeePoolBps)
	ts.Require().Lessf(totalFee, uint32(bpsDivisor),
		"%s: feeLpBps+feePoolBps must be < %d, got %d", label, bpsDivisor, totalFee)

	// Borrowed must never exceed reserves on-chain.
	if e.Reserve0.Sign() > 0 {
		ts.Require().NotEqualf(1, e.TotalBorrowed0.Cmp(e.Reserve0),
			"%s: totalBorrowed0 > reserve0 (impossible on-chain)", label)
	}
	if e.Reserve1.Sign() > 0 {
		ts.Require().NotEqualf(1, e.TotalBorrowed1.Cmp(e.Reserve1),
			"%s: totalBorrowed1 > reserve1 (impossible on-chain)", label)
	}
}

// TestCrossCheckQuoter is the strongest correctness check: for the same
// (tokenIn, tokenOut, amountIn) we ask the on-chain UniPoolQuoter and our
// off-chain simulator and require an EXACT match.
//
// Determinism: we read the pair state AND every Quoter.getAmountOut in ONE
// atomic multicall, so they all evaluate at the same block. We then read that
// block's timestamp and inject it into the simulator clock, so our VR
// projection uses the exact same instant as the on-chain quote. With state,
// timestamp and quote all aligned to one block, getAmountOut must match to the
// wei (the only thing we skip — pending liquidations — would otherwise be the
// sole source of divergence).
func (ts *PoolListTrackerIntegrationSuite) TestCrossCheckQuoter_WETHUSDT() {
	quoterABI, err := abi.JSON(bytes.NewReader(quoterABIJsonForTest))
	ts.Require().NoError(err)

	cases := []struct {
		name     string
		tokenIn  string
		tokenOut string
		amountIn *big.Int
	}{
		{"0.001 WETH -> USDT", arbWETHAddr, arbUSDTAddr, big.NewInt(1e15)},
		{"0.1 WETH -> USDT", arbWETHAddr, arbUSDTAddr, big.NewInt(1e17)},
		{"1 USDT -> WETH", arbUSDTAddr, arbWETHAddr, big.NewInt(1e6)},
		{"1000 USDT -> WETH", arbUSDTAddr, arbWETHAddr, big.NewInt(1e9)},
	}

	mc3ABI, err := abi.JSON(strings.NewReader(
		`[{"inputs":[],"name":"getCurrentBlockTimestamp","outputs":[{"type":"uint256"}],"stateMutability":"view","type":"function"}]`))
	ts.Require().NoError(err)

	var (
		blockTsBig      = new(big.Int)
		reserves        reservesABI
		vrWrap          struct{ Reserves virtualReservesABI }
		lastUpdate      = new(big.Int)
		priceDecay      = new(big.Int)
		fees            feesBpsABI
		totalBorrowed0  = new(big.Int)
		totalBorrowed1  = new(big.Int)
		swapPriceTolBps uint16
		quoterOut       = make([]*big.Int, len(cases))
	)
	reserves.Reserve0, reserves.Reserve1 = new(big.Int), new(big.Int)

	req := ts.client.NewRequest().SetContext(context.Background())
	pair := uniPoolArbWETHUSDTPair
	req.AddCall(&ethrpc.Call{ABI: mc3ABI, Target: arbitrumMulticall3, Method: "getCurrentBlockTimestamp"}, []any{&blockTsBig})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetReserves}, []any{&reserves})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetVirtualReserves}, []any{&vrWrap})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetLastUpdateTimestamp}, []any{&lastUpdate})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetPriceDecay}, []any{&priceDecay})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetFeesBps}, []any{&fees})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetTotalBorrowed0}, []any{&totalBorrowed0})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetTotalBorrowed1}, []any{&totalBorrowed1})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pair, Method: pairMethodGetSwapPriceToleranceBps}, []any{&swapPriceTolBps})
	for i := range cases {
		quoterOut[i] = new(big.Int)
		req.AddCall(&ethrpc.Call{
			ABI:    quoterABI,
			Target: uniPoolArbQuoterAddr,
			Method: "getAmountOut",
			Params: []any{common.HexToAddress(cases[i].tokenIn), common.HexToAddress(cases[i].tokenOut), cases[i].amountIn},
		}, []any{&quoterOut[i]})
	}
	_, err = req.TryBlockAndAggregate()
	ts.Require().NoError(err)

	extra := Extra{
		Reserve0:              reserves.Reserve0,
		Reserve1:              reserves.Reserve1,
		VirtualReserve0In:     vrWrap.Reserves.VirtualReserve0In,
		VirtualReserve0Out:    vrWrap.Reserves.VirtualReserve0Out,
		VirtualReserve1In:     vrWrap.Reserves.VirtualReserve1In,
		VirtualReserve1Out:    vrWrap.Reserves.VirtualReserve1Out,
		LastUpdateTimestamp:   lastUpdate.Uint64(),
		PriceDecay:            priceDecay.Uint64(),
		FeeLpBps:              fees.FeeLpBps,
		FeePoolBps:            fees.FeePoolBps,
		TotalBorrowed0:        totalBorrowed0,
		TotalBorrowed1:        totalBorrowed1,
		SwapPriceToleranceBps: swapPriceTolBps,
	}
	extraBytes, err := json.Marshal(extra)
	ts.Require().NoError(err)

	blockTs := blockTsBig.Int64()
	ts.Require().Positive(blockTs, "multicall must report a block timestamp")
	restore := nowUnix
	nowUnix = func() int64 { return blockTs }
	defer func() { nowUnix = restore }()

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  pair,
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: arbWETHAddr, Swappable: true},
			{Address: arbUSDTAddr, Swappable: true},
		},
		Reserves: entity.PoolReserves{reserves.Reserve0.String(), reserves.Reserve1.String()},
		Extra:    string(extraBytes),
	})
	ts.Require().NoError(err)

	for i, tc := range cases {
		ts.Run(tc.name, func() {
			ours, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: tc.amountIn},
				TokenOut:      tc.tokenOut,
			})
			ts.Require().NoError(err)

			diff := new(big.Int).Sub(ours.TokenAmountOut.Amount, quoterOut[i])
			ts.T().Logf("[xcheck @ts %d] %-22s simulator=%s on-chain=%s diff=%s",
				blockTs, tc.name, ours.TokenAmountOut.Amount, quoterOut[i], diff)

			ts.Require().Truef(diff.Sign() == 0,
				"%s: off-chain vs on-chain quote must match exactly (same block), diff=%s",
				tc.name, diff)
		})
	}
}

func TestPoolListTrackerIntegrationSuite(t *testing.T) {
	// NOT parallel on purpose: TestCrossCheckQuoter overrides the package-level
	// nowUnix clock. Running in the sequential phase guarantees no parallel unit
	// test observes the override (parallel tests run only after sequential ones).
	test.SkipCI(t)
	suite.Run(t, new(PoolListTrackerIntegrationSuite))
}
