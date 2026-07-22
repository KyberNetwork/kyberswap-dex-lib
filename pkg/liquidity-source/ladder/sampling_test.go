package ladder

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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

// TestEstimateFarthestProbedAmount checks that it projects the previous
// ladder's own last point (not its depletion point) by the output-reserve
// ratio, and returns nil for an empty ladder.
func TestEstimateFarthestProbedAmount(t *testing.T) {
	t.Parallel()

	prevReserve1 := big.NewInt(100)
	currentReserve1 := big.NewInt(1000) // 10x growth

	prevLadder := []Point{{100, 20}, {200, 38}, {5000, 97}}
	got := EstimateFarthestProbedAmount(prevLadder, prevReserve1, currentReserve1)
	if assert.NotNil(t, got) {
		// last point's amountIn (5000) scaled by the 10x reserve ratio.
		assert.Equal(t, int64(50_000), got.Int64())
	}

	assert.Nil(t, EstimateFarthestProbedAmount(nil, prevReserve1, currentReserve1))
	assert.Nil(t, EstimateFarthestProbedAmount(prevLadder, nil, currentReserve1))
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

// TestSamplePoints_GuidedByPreviousLadder checks that SamplePoints prefers
// the previous cycle's ladder (via EstimateNearCapacityAmount) over the raw
// input-side reserve once a previous state is available, and falls back to
// the input-side reserve on the very first probe (no previous state yet).
// Any ladder-quoted pool tracker (liquidcore, caliberprop, ...) can rely on
// this shared behavior instead of reimplementing it.
func TestSamplePoints_GuidedByPreviousLadder(t *testing.T) {
	t.Parallel()

	// A previous ladder for direction 0 (token0->token1) whose marginal
	// rate of return clearly drops (DepletionAmountIn flags it at
	// amountIn=300, where the cumulative drop from the best rate (0.2, at
	// amountIn=100) first reaches rateDropFraction: 0.16/0.2 = 0.8), but
	// which was still actually successfully probed all the way out to 5000
	// (its last point) despite the flattened rate after 300.
	prevLadder0 := []Point{
		{100, 20}, {200, 38}, {300, 54}, {400, 68}, {500, 96}, {5000, 97},
	}
	extra, err := json.Marshal(Extra{Ladders: [2][]Point{prevLadder0, nil}})
	assert.NoError(t, err)

	t.Run("first probe: no previous state, falls back to input reserve", func(t *testing.T) {
		t.Parallel()
		p := entity.Pool{} // Extra empty, Reserves empty
		points := SamplePoints(p, 0, big.NewInt(1_000_000), big.NewInt(100))
		assert.NotEmpty(t, points)
		// with no guidance, top of the grid should track the input reserve
		// (1_000_000 * 99% via BuildSamplePoints), not the tiny prevLadder scale.
		last := points[len(points)-1]
		assert.Greater(t, last.Int64(), int64(500_000))
	})

	t.Run("subsequent probe: guided by previous ladder, scaled by reserve1 change", func(t *testing.T) {
		t.Parallel()
		p := entity.Pool{
			Extra:    string(extra),
			Reserves: entity.PoolReserves{"999999", "100"}, // prev reserve0 unused here, prev reserve1=100
		}
		// current reserve1 has grown 10x since the previous cycle; current
		// input reserve0 is huge and unbalanced (would badly overstate the
		// tradeable range if used directly).
		points := SamplePoints(p, 0, big.NewInt(1_000_000_000), big.NewInt(1000))
		assert.NotEmpty(t, points)
		last := points[len(points)-1]
		// EstimateFarthestProbedAmount wins here: prevLadder0's own last
		// point (5000) scaled by the reserve1 ratio (1000/100=10) = 50000,
		// past the near-cap-based growth canaries (3000*4=12000) but still
		// nowhere near the unbalanced 1_000_000_000 input reserve -- the
		// pool stays quotable out to its full previously-probed extent even
		// though the dense/accurate part of the grid stops much earlier.
		assert.Equal(t, int64(50_000), last.Int64())
	})
}

// TestWithGrowthCanaries guards the fix for a one-way ratchet:
// DepletionAmountIn's rate-drop signal alone can't distinguish a genuine
// depletion cliff from a plain constant-product pool's ordinary curvature
// (a coarse geometric grid's last segment crosses a large relative rate
// drop purely from that curvature), so once a guided range undershoots the
// real capacity, the next cycle's rescaled grid re-triggers the same false
// "depletion" call and the range keeps shrinking with no way back. Canaries
// scaled off the current ceiling -- not the raw reserve -- ride along in
// the same probe batch and give DepletionAmountIn a chance to see past that
// ceiling every cycle, so a range that undershot has a path to recover.
func TestWithGrowthCanaries(t *testing.T) {
	t.Parallel()

	t.Run("appends canaries scaled off nearCap, not the raw reserve", func(t *testing.T) {
		t.Parallel()
		nearCap := big.NewInt(1000)
		reserve := big.NewInt(1_000_000_000) // far larger than any canary should reach
		got := withGrowthCanaries([]*big.Int{big.NewInt(500), big.NewInt(1000)}, nearCap, reserve)
		want := []int64{500, 1000, 2000, 4000} // nearCap*2, nearCap*4 appended
		if assert.Len(t, got, len(want)) {
			for i, w := range want {
				assert.Equal(t, w, got[i].Int64())
			}
		}
	})

	t.Run("clamps canaries at the raw reserve and dedupes", func(t *testing.T) {
		t.Parallel()
		nearCap := big.NewInt(600_000) // *2 and *4 both exceed reserve below
		reserve := big.NewInt(1_000_000)
		got := withGrowthCanaries([]*big.Int{big.NewInt(300_000)}, nearCap, reserve)
		want := []int64{300_000, 1_000_000} // both canaries clamp to reserve and dedupe into one
		if assert.Len(t, got, len(want)) {
			for i, w := range want {
				assert.Equal(t, w, got[i].Int64())
			}
		}
	})
}
