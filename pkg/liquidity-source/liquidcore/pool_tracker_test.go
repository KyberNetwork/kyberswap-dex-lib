package liquidcore

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// TestSamplePoints_GuidedByPreviousLadder checks that samplePoints prefers
// the previous cycle's ladder (via estimateNearCapacityAmount) over the raw
// input-side reserve once a previous state is available, and falls back to
// the input-side reserve on the very first probe (no previous state yet).
func TestSamplePoints_GuidedByPreviousLadder(t *testing.T) {
	t.Parallel()

	tracker := &PoolTracker{}

	// A previous ladder for direction 0 (token0->token1) that clearly
	// reached depletion of its cycle's reserve1 (100) by amountIn=500.
	prevLadder0 := []ladder.Point{
		{100, 20}, {200, 38}, {300, 54}, {400, 68}, {500, 96},
	}
	extra, err := json.Marshal(ladder.Extra{Ladders: [2][]ladder.Point{prevLadder0, nil}})
	assert.NoError(t, err)

	t.Run("first probe: no previous state, falls back to input reserve", func(t *testing.T) {
		t.Parallel()
		p := entity.Pool{} // Extra empty, Reserves empty
		points := tracker.samplePoints(p, 0, big.NewInt(1_000_000), big.NewInt(100))
		assert.NotEmpty(t, points)
		// with no guidance, top of the grid should track the input reserve
		// (1_000_000 * 99% via geometricBps), not the tiny prevLadder scale.
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
		points := tracker.samplePoints(p, 0, big.NewInt(1_000_000_000), big.NewInt(1000))
		assert.NotEmpty(t, points)
		last := points[len(points)-1]
		// previous near-cap input (500) scaled by reserve1 ratio (1000/100=10) = 5000,
		// not anywhere near the unbalanced 1_000_000_000 input reserve.
		assert.Less(t, last.Int64(), int64(50_000))
	})
}
