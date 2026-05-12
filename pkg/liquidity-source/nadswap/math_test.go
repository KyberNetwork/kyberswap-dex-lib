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

// Meme buy: tokenIn = quoteToken
// reserveQuote=10000, reserveBase=10000, amountIn=1000, feeRate=100 (=1%)
// totalFeeRate = 25 + 100 = 125
// amountInWithFee = 1000 * (10000 - 125) = 9_875_000
// amountOut = 9_875_000 * 10000 / (10000 * 10000 + 9_875_000) = 98_750_000_000 / 109_875_000 = 898
func TestGetAmountOut_MemeBuy(t *testing.T) {
	t.Parallel()
	out, err := getAmountOutMemeBuy(u("1000"), u("10000"), u("10000"), 100)
	require.NoError(t, err)
	assert.Equal(t, "898", out.Dec())
}

func TestGetAmountOut_MemeBuy_InvalidFeeRate(t *testing.T) {
	t.Parallel()
	_, err := getAmountOutMemeBuy(u("1000"), u("10000"), u("10000"), 9976)
	assert.ErrorIs(t, err, ErrInvalidFeeRate)
}

// Meme sell: tokenIn = baseToken
// reserveBase=10000, reserveQuote=10000, amountIn=1000, feeRate=100
// amountInWithLpFee = 1000 * (10000 - 25) = 9_975_000
// gross = 9_975_000 * 10000 / (10000*10000 + 9_975_000) = 907
// swapFee = ceil(907 * 100 / (10000 - 25)) = ceil(90700 / 9975) = ceil(9.0927...) = 10
// net = 907 - 10 = 897
func TestGetAmountOut_MemeSell(t *testing.T) {
	t.Parallel()
	out, err := getAmountOutMemeSell(u("1000"), u("10000"), u("10000"), 100)
	require.NoError(t, err)
	assert.Equal(t, "897", out.Dec())
}

// Sell with feeRate=0 must equal the general-pair LP-only result.
func TestGetAmountOut_MemeSell_ZeroFee_EqualsGeneral(t *testing.T) {
	t.Parallel()
	memeOut, err := getAmountOutMemeSell(u("1000"), u("10000"), u("10000"), 0)
	require.NoError(t, err)
	genOut, err := getAmountOutGeneral(u("1000"), u("10000"), u("10000"))
	require.NoError(t, err)
	assert.Equal(t, genOut.Dec(), memeOut.Dec())
}

// Round-trip property: getAmountIn(getAmountOut(x)) >= x, and the gap is within
// at most 1 unit (rounding). Asserts the exact-output formula is the inverse.
func TestRoundTrip_GeneralPair(t *testing.T) {
	t.Parallel()
	out, err := getAmountOutGeneral(u("1000"), u("10000"), u("10000"))
	require.NoError(t, err)
	in, err := getAmountInGeneral(out, u("10000"), u("10000"))
	require.NoError(t, err)
	// in should be >= 1000 (we may slightly over-collect due to ceil)
	diff := new(uint256.Int).Sub(in, u("1000"))
	assert.True(t, diff.LtUint64(2), "diff=%s", diff.Dec())
}

func TestRoundTrip_MemeBuy(t *testing.T) {
	t.Parallel()
	out, err := getAmountOutMemeBuy(u("1000"), u("10000"), u("10000"), 100)
	require.NoError(t, err)
	in, err := getAmountInMemeBuy(out, u("10000"), u("10000"), 100)
	require.NoError(t, err)
	diff := new(uint256.Int).Sub(in, u("1000"))
	assert.True(t, diff.LtUint64(2), "diff=%s", diff.Dec())
}

func TestRoundTrip_MemeSell(t *testing.T) {
	t.Parallel()
	out, err := getAmountOutMemeSell(u("1000"), u("10000"), u("10000"), 100)
	require.NoError(t, err)
	in, err := getAmountInMemeSell(out, u("10000"), u("10000"), 100)
	require.NoError(t, err)
	diff := new(uint256.Int).Sub(in, u("1000"))
	assert.True(t, diff.LtUint64(2), "diff=%s", diff.Dec())
}

func TestGetAmountIn_ZeroOutput(t *testing.T) {
	t.Parallel()
	_, err := getAmountInGeneral(u("0"), u("10000"), u("10000"))
	assert.ErrorIs(t, err, ErrInsufficientOutput)
	_, err = getAmountInMemeBuy(u("0"), u("10000"), u("10000"), 100)
	assert.ErrorIs(t, err, ErrInsufficientOutput)
	_, err = getAmountInMemeSell(u("0"), u("10000"), u("10000"), 100)
	assert.ErrorIs(t, err, ErrInsufficientOutput)
}

func TestGetAmountIn_OutputExceedsReserve(t *testing.T) {
	t.Parallel()
	_, err := getAmountInGeneral(u("10000"), u("10000"), u("10000"))
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}
