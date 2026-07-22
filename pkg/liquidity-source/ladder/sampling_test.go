package ladder

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEstimateNearCapacityAmount_Found uses liquidcoreUSDCkHYPELadder (block
// 41068712, reserve1=347630887026632391957 at capture time) as the
// "previous cycle" and checks that a later cycle's much-smaller reserve1
// (observed at block 41113644) produces a proportionally scaled-down
// estimate rather than reserve0's unrelated scale.
func TestEstimateNearCapacityAmount_Found(t *testing.T) {
	t.Parallel()

	prevReserve1 := big.NewInt(0)
	prevReserve1.SetString("347630887026632391957", 10)
	currentReserve1 := big.NewInt(0)
	currentReserve1.SetString("258197876761440418668", 10)

	got := EstimateNearCapacityAmount(liquidcoreUSDCkHYPELadder, prevReserve1, currentReserve1)
	if assert.NotNil(t, got) {
		// previous near-cap point was liquidcoreUSDCkHYPELadder's amountIn
		// at 27987096900 (the first point whose amountOut reaches 95% of
		// prevReserve1); scaled by currentReserve1/prevReserve1 (~0.743).
		wantRatio := 258197876761440418668.0 / 347630887026632391957.0
		wantAmount := 27987096900.0 * wantRatio
		gotF, _ := got.Float64()
		assert.InEpsilonf(t, wantAmount, gotF, 1e-6, "got %v want ~%v", got, wantAmount)
	}
}

// TestEstimateNearCapacityAmount_NotFound guards the fallback path: a
// ladder that never gets close to depleting its cycle's reserve (all points
// well under nearCapacityFraction of reserve1) has nothing useful to guide
// from, so callers should fall back to their default reserve-based basis.
func TestEstimateNearCapacityAmount_NotFound(t *testing.T) {
	t.Parallel()

	farFromDepleted := []Point{
		{1000, 10},
		{2000, 19},
		{3000, 27},
	}
	prevReserve1 := big.NewInt(1_000_000) // ladder's max output (27) is nowhere near 95% of this
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
		// the final, exact-pinned entry in geometricBps).
		assert.InEpsilon(t, 20_000_000_000.0, float64(last.Int64()), 1e-6)
	}

	assert.Nil(t, BuildSamplePointsFrom(nil, SampleSize))
	assert.Nil(t, BuildSamplePointsFrom(big.NewInt(0), SampleSize))
}
