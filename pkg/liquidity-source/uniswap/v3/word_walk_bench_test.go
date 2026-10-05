package uniswapv3

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// deepFullRangePool is a spacing-1 pool at tick 0 holding a 1e21 full-range position plus a 1e23
// position over [-600, 600]: large swaps drain the concentrated range, then walk hundreds of empty
// words at full-range liquidity. This is the shape router probes hit on ordinary mainnet pools.
func deepFullRangePool(tb testing.TB) *Pool {
	p, err := NewPool(500, sqrtAt(&testing.T{}, 0), *uint256.MustFromDecimal("100001000000000000000000"), 0,
		fullRangeTicks(1, "1000000000000000000000",
			TickU256{Index: -600, LiquidityGross: uint256.MustFromDecimal("100000000000000000000000"),
				LiquidityNet: int256.MustFromDec("100000000000000000000000")},
			TickU256{Index: 600, LiquidityGross: uint256.MustFromDecimal("100000000000000000000000"),
				LiquidityNet: int256.MustFromDec("-100000000000000000000000")}), 1)
	if err != nil {
		tb.Fatal(err)
	}
	return p
}

// gapPool is a spacing-1 pool at tick 0 with positions over [-600, 600] and [-200000, -190000]
// only: a large zero-for-one swap drains the first, then walks ~740 words at zero liquidity.
func gapPool(tb testing.TB) *Pool {
	l := "100000000000000000000000"
	tick := func(i int, net string) TickU256 {
		return TickU256{Index: i, LiquidityGross: uint256.MustFromDecimal(l), LiquidityNet: int256.MustFromDec(net)}
	}
	p, err := NewPool(500, sqrtAt(&testing.T{}, 0), *uint256.MustFromDecimal(l), 0, []TickU256{
		tick(-200000, l), tick(-190000, "-"+l), tick(-600, l), tick(600, "-"+l)}, 1)
	if err != nil {
		tb.Fatal(err)
	}
	return p
}

func benchmarkSwap(b *testing.B, p *Pool, zeroForOne bool, amount string) {
	a := signed(amount)
	r, err := p.Swap(zeroForOne, a, uint256.Int{})
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		_, _ = p.Swap(zeroForOne, a, uint256.Int{})
	}
	b.ReportMetric(float64(r.CrossEmptyWordLoops), "words")
}

func BenchmarkSwapDeepFullRangeWalk(b *testing.B) {
	benchmarkSwap(b, deepFullRangePool(b), true, "1000000000000000000000000")
}

func BenchmarkSwapDeepFullRangeWalkUp(b *testing.B) {
	benchmarkSwap(b, deepFullRangePool(b), false, "1000000000000000000000000")
}

func BenchmarkSwapDeepFullRangeWalkExactOut(b *testing.B) {
	benchmarkSwap(b, deepFullRangePool(b), true, "-900000000000000000000000")
}

func BenchmarkSwapZeroLiquidityGap(b *testing.B) {
	benchmarkSwap(b, gapPool(b), true, "10000000000000000000000")
}

// TestCalcAmountOutResultsDoNotAlias guards the single-allocation result backing: callers may mutate
// a returned amount, which must not reach another quote or the pool state.
func TestCalcAmountOutResultsDoNotAlias(t *testing.T) {
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: bignumber.NewBig10("10000000000000000000")},
		TokenOut: "0xf1b99e3e573a1a9c5e6b2ce818b617f0e664e86b",
	}
	sim := mustNewPoolSimulatorBench()
	sqrtBefore := sim.V3Pool.SqrtRatioX96
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	want := new(big.Int).Set(first.TokenAmountOut.Amount)

	first.TokenAmountOut.Amount.Add(first.TokenAmountOut.Amount, big.NewInt(1))
	first.SwapInfo.(SwapInfo).NextStateSqrtRatioX96.SetOne()
	require.Equal(t, sqrtBefore, sim.V3Pool.SqrtRatioX96)

	second, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, want, second.TokenAmountOut.Amount)
	require.Equal(t, 0, second.RemainingTokenAmountIn.Amount.Sign())
}

// BenchmarkGetTickAtSqrtRatioInterior uses prices inside a tick, the shape a partial swap step
// leaves; BenchmarkGetTickAtSqrtRatio's exact tick prices always take the verification path.
func BenchmarkGetTickAtSqrtRatioInterior(b *testing.B) {
	var prices []uint256.Int
	for _, tick := range []int{0, 1000, 10000, 27139, -1000, -27139} {
		var at, next uint256.Int
		_ = GetSqrtRatioAtTick(tick, &at)
		_ = GetSqrtRatioAtTick(tick+1, &next)
		prices = append(prices, *next.Add(&at, &next).Rsh(&next, 1))
	}
	i := 0
	for b.Loop() {
		_, _ = GetTickAtSqrtRatio(&prices[i%len(prices)])
		i++
	}
}
