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
