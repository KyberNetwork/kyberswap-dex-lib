package lunarbase

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func u(s string) *uint256.Int {
	v, _ := uint256.FromDecimal(s)
	return v
}

func TestIsqrt(t *testing.T) {
	cases := []struct {
		input, expected uint64
	}{
		{0, 0},
		{1, 1},
		{4, 2},
		{9, 3},
		{10, 3},
		{100, 10},
	}
	for _, tc := range cases {
		got := isqrt(uint256.NewInt(tc.input))
		assert.Equal(t, uint256.NewInt(tc.expected), got, "isqrt(%d)", tc.input)
	}
}

func TestConcentrationQ48_ZeroAmount(t *testing.T) {
	var c uint256.Int
	concentrationQ48(&c, uint256.NewInt(1<<48), new(uint256.Int),
		uint256.NewInt(10_000), uint256.NewInt(10_000), 5000, true)
	assert.True(t, c.IsZero(), "amountIn=0 should yield c=0 (linear fallback path)")
}

func TestConcentrationQ48_ZeroK(t *testing.T) {
	var c uint256.Int
	concentrationQ48(&c, uint256.NewInt(1<<48), uint256.NewInt(1000),
		uint256.NewInt(10_000), uint256.NewInt(10_000), 0, true)
	assert.True(t, c.IsZero(), "K=0 should yield c=0 (linear fallback path)")
}

func TestConcentrationQ48_SaturatesAtQ48(t *testing.T) {
	// amountInWealth ≥ totalWealth → r = 1 → c = K (no saturation), but
	// stored K is Q20.12 so the effective multiplier scales by /Q12.
	var c uint256.Int
	concentrationQ48(&c, uint256.NewInt(1<<48), uint256.NewInt(1_000_000_000_000),
		uint256.NewInt(1), uint256.NewInt(1), ^uint32(0), false)
	assert.True(t, c.Eq(q48) || c.Lt(q48), "c must never exceed Q48")
}

func TestQuoteReturnsZeroWhenNoLiquidity(t *testing.T) {
	params := &PoolParams{
		SqrtPriceX48:   uint256.NewInt(1 << 48),
		FeeAskX24:      0,
		FeeBidX24:      1 << 20, // ~6.25% in Q24
		ReserveX:       new(uint256.Int),
		ReserveY:       new(uint256.Int),
		ConcentrationK: 5000,
	}
	result := quoteXToY(params, uint256.NewInt(1000))
	assert.True(t, result.AmountOut.IsZero())
}
