package ladder

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEstimateNearCapacityAmount_Found uses liquidcoreUSDCkHYPELadder (block
// 41125109, reserve1=260569475582118373524 at capture time) as the
// "previous cycle" and checks that a later cycle's smaller reserve1
// produces a proportionally scaled-down estimate rather than reserve0's
// unrelated scale.
func TestEstimateNearCapacityAmount_Found(t *testing.T) {
	t.Parallel()

	prevReserve1 := big.NewInt(0)
	prevReserve1.SetString("260569475582118373524", 10)
	currentReserve1 := big.NewInt(0)
	currentReserve1.SetString("183804153372913102996", 10)

	got := EstimateNearCapacityAmount(liquidcoreUSDCkHYPELadder, prevReserve1, currentReserve1)
	if assert.NotNil(t, got) {
		// previous near-cap point was liquidcoreUSDCkHYPELadder's amountIn
		// at 17615352630 (the first point where DepletionAmountIn sees the
		// marginal rate drop to rateDropFraction of the best rate seen);
		// scaled by currentReserve1/prevReserve1 (~0.7054).
		wantRatio := 183804153372913102996.0 / 260569475582118373524.0
		wantAmount := 17615352630.0 * wantRatio
		gotF, _ := got.Float64()
		assert.InEpsilonf(t, wantAmount, gotF, 1e-6, "got %v want ~%v", got, wantAmount)
	}
}

// TestEstimateNearCapacityAmount_NotFound guards the fallback path: a
// ladder whose marginal rate only ever declines gently (ordinary slippage,
// never a real depletion cliff) has nothing useful to guide from, so callers
// should fall back to their default reserve-based basis.
func TestEstimateNearCapacityAmount_NotFound(t *testing.T) {
	t.Parallel()

	farFromDepleted := []Point{
		{1000, 10},
		{2000, 19.9},
		{3000, 29.7},
	}
	prevReserve1 := big.NewInt(1_000_000)
	currentReserve1 := big.NewInt(900_000)

	got := EstimateNearCapacityAmount(farFromDepleted, prevReserve1, currentReserve1)
	assert.Nil(t, got)
}

func TestEstimateNearCapacityAmount_NilInputs(t *testing.T) {
	t.Parallel()

	assert.Nil(t, EstimateNearCapacityAmount(liquidcoreUSDCkHYPELadder, nil, big.NewInt(1)))
	assert.Nil(t, EstimateNearCapacityAmount(liquidcoreUSDCkHYPELadder, big.NewInt(1), nil))
	assert.Nil(t, EstimateNearCapacityAmount(liquidcoreUSDCkHYPELadder, big.NewInt(0), big.NewInt(1)))
}

// TestBuildSamplePointsFrom checks that the grid it produces is anchored so
// its top point lands at nearCapacityAmount (the sampleBpsMax point),
// matching BuildSamplePointsN's own shape.
func TestBuildSamplePointsFrom(t *testing.T) {
	t.Parallel()

	nearCap := big.NewInt(20_000_000_000)
	got := BuildSamplePointsFrom(nearCap, SampleSize)
	if assert.NotEmpty(t, got) {
		last := got[len(got)-1]
		// last point should be very close to nearCap itself (bps=9900 is
		// the final, exact-pinned entry in dgeoBps).
		assert.InEpsilon(t, 20_000_000_000.0, float64(last.Int64()), 1e-6)
	}

	assert.Nil(t, BuildSamplePointsFrom(nil, SampleSize))
	assert.Nil(t, BuildSamplePointsFrom(big.NewInt(0), SampleSize))
}
