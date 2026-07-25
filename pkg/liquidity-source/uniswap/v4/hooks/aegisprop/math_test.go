package aegisprop

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

// price expressed directly in token1-per-token0 * 1e18 units, built from a human-readable multiplier.
func priceLevel(priceMultiplier, amplitude uint64) Level {
	price := new(uint256.Int).Mul(uint256.NewInt(priceMultiplier), uPriceScale)
	return Level{Price: price, Amplitude: uint256.NewInt(amplitude)}
}

func TestWalkExactIn_Bids_SpansMultipleLevels(t *testing.T) {
	// price 2000 (best) then 1990, matching integration guide §4 B3 semantics for the bid side.
	levels := []Level{priceLevel(2000, 100), priceLevel(1990, 50)}

	out, updated, filled := walkExactIn(levels, uint256.NewInt(120), true)

	assert.True(t, filled)
	// 100 * 2000 + 20 * 1990 = 200000 + 39800 = 239800
	assert.Equal(t, "239800", out.Dec())
	assert.Equal(t, "0", updated[0].Amplitude.Dec())
	assert.Equal(t, "30", updated[1].Amplitude.Dec())
}

func TestWalkExactIn_Asks_ConvertsThroughPrice(t *testing.T) {
	levels := []Level{priceLevel(2, 100)} // cap in token1 = 100 * 2 = 200

	out, updated, filled := walkExactIn(levels, uint256.NewInt(150), false)

	assert.True(t, filled)
	// 150 token1 in / price(2) = 75 token0 out
	assert.Equal(t, "75", out.Dec())
	assert.Equal(t, "25", updated[0].Amplitude.Dec())
}

func TestWalkExactIn_InsufficientDepth(t *testing.T) {
	levels := []Level{priceLevel(2000, 100)}

	_, _, filled := walkExactIn(levels, uint256.NewInt(150), true)

	assert.False(t, filled)
}

func TestWalkExactOut_Bids_MatchesExactInInverse(t *testing.T) {
	levels := []Level{priceLevel(2, 100)}

	in, updated, filled := walkExactOut(levels, uint256.NewInt(100), true)

	assert.True(t, filled)
	assert.Equal(t, "50", in.Dec())
	assert.Equal(t, "50", updated[0].Amplitude.Dec())
}

func TestWalkExactOut_Asks_MatchesExactInInverse(t *testing.T) {
	levels := []Level{priceLevel(2, 100)}

	in, updated, filled := walkExactOut(levels, uint256.NewInt(75), false)

	assert.True(t, filled)
	assert.Equal(t, "150", in.Dec())
	assert.Equal(t, "25", updated[0].Amplitude.Dec())
}
