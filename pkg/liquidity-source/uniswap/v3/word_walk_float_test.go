package uniswapv3

import (
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestWalkWordsFloatTolerance checks the float word walk against the contract-loop oracle on swaps
// that walk well past floatWalkAfterWords. The router tolerates 0.05bps of quote error, and a quote
// must never beat the contract: exact-in may only return less, exact-out may only charge more.
func TestWalkWordsFloatTolerance(t *testing.T) {
	t.Parallel()

	dustUp := func(t *testing.T) *Pool {
		p, err := NewPool(100, sqrtAt(t, -291970), *uint256.NewInt(4787383), -291970,
			fullRangeTicks(1, "4787383"), 1)
		require.NoError(t, err)
		return p
	}
	for _, tc := range []struct {
		name       string
		pool       func(*testing.T) *Pool
		zeroForOne bool
		amounts    []string
	}{
		{"deep down", func(t *testing.T) *Pool { return deepFullRangePool(t) }, true,
			[]string{"300000000000000000000000", "1000000000000000000000000", "-2956301085000000000000", "-2956301086900000000000",
				"-900000000000000000000000"}},
		{"deep up", func(t *testing.T) *Pool { return deepFullRangePool(t) }, false,
			[]string{"300000000000000000000000", "1000000000000000000000000", "-2956301085000000000000", "-2956301086900000000000",
				"-900000000000000000000000"}},
		{"dust down", func(t *testing.T) *Pool { return avaxDustPool(t) }, true,
			[]string{"300", "1000", "100000", "-10000000000000"}},
		{"dust up", dustUp, false, []string{"300", "1000", "100000", "-10466000000000"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, amount := range tc.amounts {
				want, err := newRefPool(tc.pool(t)).swap(tc.zeroForOne, signed(amount))
				require.NoError(t, err)
				got, err := tc.pool(t).Swap(tc.zeroForOne, signed(amount), uint256.Int{})
				require.NoError(t, err)
				require.Greater(t, want.CrossEmptyWordLoops, floatWalkAfterWords, amount)

				var w, g uint256.Int
				w.Abs(&want.AmountCalculated)
				g.Abs(&got.AmountCalculated)
				wf, gf := w.Float64(), g.Float64()
				t.Logf("%s: want %s got %s words %d/%d diff %.3gbps", amount, w.Dec(), g.Dec(),
					want.CrossEmptyWordLoops, got.CrossEmptyWordLoops, (gf-wf)/wf*1e4)
				if amount[0] == '-' {
					require.False(t, g.Lt(&w), "exact-out must not charge less than the contract")
				} else {
					require.False(t, g.Gt(&w), "exact-in must not return more than the contract")
				}
				require.InDelta(t, 0, (gf-wf)/wf*1e4, 0.05, "quote drifts beyond 0.05bps")
				require.Equal(t, want.RemainingAmountIn.IsZero(), got.RemainingAmountIn.IsZero())
			}
		})
	}
}

// TestCeilFeeExact pins ceilFee to integer ceil(in*fee/feeDenom) wherever it claims exactness,
// across all fee tiers: its reciprocal estimate is corrected by one in either direction.
func TestCeilFeeExact(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(7, 8))
	for i := range 1000000 {
		fee := rng.Uint64N(maxFeeInt)
		if i%3 == 0 {
			fee = []uint64{1, 100, 500, 3000, 10000, 999999}[i%6]
		}
		denom := maxFeeInt - fee
		in := rng.Uint64N(1<<52/max(fee, 1)) + 1
		if i%5 == 0 { // exact multiples land on the boundary the correction guards
			in = (in/denom + 1) * denom
			in = min(in, (1<<52-1)/max(fee, 1))
		}
		want := (in*fee + denom - 1) / denom
		got := ceilFee(float64(in), float64(fee), float64(denom), 1/float64(denom))
		require.Equal(t, float64(want), got, "in %d fee %d", in, fee)
	}
}

// TestZeroLiquidityJumpMatchesContract: at zero liquidity every word's amounts are zero, so jumping
// the run must reproduce the contract loop exactly, word count included. Each swap walks fewer than
// floatWalkAfterWords funded words, so the float walk stays out of it.
func TestZeroLiquidityJumpMatchesContract(t *testing.T) {
	t.Parallel()

	for _, amount := range []string{"10000000000000000000000", "100000000000000000000000", "-2956000000000000000000"} {
		want, err := newRefPool(gapPool(t)).swap(true, signed(amount))
		require.NoError(t, err)
		require.Greater(t, want.CrossEmptyWordLoops, 700, amount)
		got, err := gapPool(t).Swap(true, signed(amount), uint256.Int{})
		require.NoError(t, err)
		require.Equal(t, want, got, amount)
	}
}

// TestWalkWordsFloatOvershootStepsBack: the exact-in swap of 572857042997820927 into the Base
// aerodrome-cl USDC/USDbC pool 0x98c7a233, one full-range position of 6694992424 liquidity, reaches
// tick -4097 on a word edge with 572856965202003819 left, and the next 1411 empty words cost
// 572856965202003968 in float: 149 more than is left, as the float remainder rounds past the exact
// one. The walk then gave up, the exact loop took one word and the walk started over, 172 times
// over the run. It now walks one word fewer, which fits, and the contract walking those words takes
// no more input and pays no less.
func TestWalkWordsFloatOvershootStepsBack(t *testing.T) {
	t.Parallel()

	const liquidity, tick = "6694992424", -4097
	start := *uint256.MustFromDecimal("64556580881331167221767657720")
	remaining := *uint256.MustFromDecimal("572856965202003819")
	// start is the edge of tick -4096; a swap that crossed down to it stands at tick -4097
	p, err := NewPool(100, start, *uint256.MustFromDecimal(liquidity), tick+1, fullRangeTicks(1, liquidity), 1)
	require.NoError(t, err)

	l, price, rem := p.Liquidity, start, remaining
	var calculated uint256.Int
	_, words, err := p.walkWordsFloat(true, true, tick, -887272, &l, MinSqrtRatioU256P1, &price, &rem,
		&calculated)
	require.NoError(t, err)
	require.Equal(t, 1410, words)
	require.False(t, rem.Gt(&remaining), "the walk spends no more than is left")

	var spent uint256.Int
	spent.Sub(&remaining, &rem)
	want, err := newRefPool(p).swap(true, spent)
	require.NoError(t, err)
	var got, paid uint256.Int
	got.Neg(&calculated)
	paid.Neg(&want.AmountCalculated)
	require.False(t, got.Gt(&paid), "the float walk pays no more than the contract")
	require.False(t, want.SqrtRatioX96.Gt(&price), "the contract reaches the walk's word edge")
}
