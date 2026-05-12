package nadswap

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

// General pair = LP fee only, no creator/protocol fee, symmetric.
// Same formula as Uniswap V2 with feeRate = 25 BPS.
//
// reserveIn=10000, reserveOut=10000, amountIn=1000
// amountInAfterLpFee = 1000 * 9975 = 9_975_000
// amountOut = 9_975_000 * 10000 / (10000*10000 + 9_975_000)
//           = 99_750_000_000 / 109_975_000
//           = 907
func TestGetAmountOut_GeneralPair_LPOnly(t *testing.T) {
	t.Parallel()
	out, err := getAmountOutGeneral(u("1000"), u("10000"), u("10000"))
	require.NoError(t, err)
	assert.Equal(t, "907", out.Dec())
}

func TestGetAmountOut_GeneralPair_Guards(t *testing.T) {
	t.Parallel()
	_, err := getAmountOutGeneral(u("0"), u("10000"), u("10000"))
	assert.ErrorIs(t, err, ErrInsufficientInput)

	_, err = getAmountOutGeneral(u("1000"), u("0"), u("10000"))
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)

	_, err = getAmountOutGeneral(u("1000"), u("10000"), u("0"))
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}
