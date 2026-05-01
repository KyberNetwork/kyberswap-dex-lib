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

// In all vectors anchorPX48 == pX48 (the on-chain `upd` resets both to the same value).
func assertXToY(t *testing.T, name string,
	pX48 string, fee uint64, resX, resY string, k uint32, dx string,
	expectedDy, expectedPNext, expectedFee string,
) {
	t.Helper()
	p := u(pX48)
	params := &PoolParams{
		SqrtPriceX48:       p,
		AnchorSqrtPriceX48: p,
		FeeQ48:             fee,
		ReserveX:           u(resX),
		ReserveY:           u(resY),
		ConcentrationK:     k,
	}
	result := quoteXToY(params, u(dx))
	assert.Equal(t, expectedDy, result.AmountOut.Dec(), "%s: dy mismatch", name)
	assert.Equal(t, expectedPNext, result.SqrtPriceNext.Dec(), "%s: pNext mismatch", name)
	assert.Equal(t, expectedFee, result.Fee.Dec(), "%s: fee mismatch", name)
}

func assertYToX(t *testing.T, name string,
	pX48 string, fee uint64, resX, resY string, k uint32, dy string,
	expectedDx, expectedPNext, expectedFee string,
) {
	t.Helper()
	p := u(pX48)
	params := &PoolParams{
		SqrtPriceX48:       p,
		AnchorSqrtPriceX48: p,
		FeeQ48:             fee,
		ReserveX:           u(resX),
		ReserveY:           u(resY),
		ConcentrationK:     k,
	}
	result := quoteYToX(params, u(dy))
	assert.Equal(t, expectedDx, result.AmountOut.Dec(), "%s: dx mismatch", name)
	assert.Equal(t, expectedPNext, result.SqrtPriceNext.Dec(), "%s: pNext mismatch", name)
	assert.Equal(t, expectedFee, result.Fee.Dec(), "%s: fee mismatch", name)
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

func TestConcentrationQ48_ZeroFee(t *testing.T) {
	c := concentrationQ48(uint256.NewInt(1<<48), 0, uint256.NewInt(1000), uint256.NewInt(10000), uint256.NewInt(10000), 5000, true)
	assert.True(t, c.IsZero())
}

func TestConcentrationQ48_ZeroAmount(t *testing.T) {
	c := concentrationQ48(uint256.NewInt(1<<48), 1000, new(uint256.Int), uint256.NewInt(10000), uint256.NewInt(10000), 5000, true)
	assert.Equal(t, uint256.NewInt(1000), c)
}

func TestQuoteReturnsZeroWhenNoLiquidity(t *testing.T) {
	p := uint256.NewInt(1 << 48)
	params := &PoolParams{
		SqrtPriceX48:       p,
		AnchorSqrtPriceX48: p,
		FeeQ48:             1 << 44,
		ReserveX:           new(uint256.Int),
		ReserveY:           new(uint256.Int),
		ConcentrationK:     5000,
	}
	result := quoteXToY(params, uint256.NewInt(1000))
	assert.True(t, result.AmountOut.IsZero())
}
