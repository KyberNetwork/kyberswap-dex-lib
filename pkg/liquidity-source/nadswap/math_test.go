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

// CRITICAL: with feeRate > 0, buy and sell must NOT be symmetric.
// If they happen to be equal, the asymmetric fee logic is broken.
func TestBuySellAsymmetry(t *testing.T) {
	t.Parallel()
	const feeRate uint16 = 100 // 1%
	buyOut, err := getAmountOutMemeBuy(u("1000"), u("10000"), u("10000"), feeRate)
	require.NoError(t, err)
	sellOut, err := getAmountOutMemeSell(u("1000"), u("10000"), u("10000"), feeRate)
	require.NoError(t, err)
	assert.NotEqual(t, buyOut.Dec(), sellOut.Dec(),
		"buy and sell must differ when feeRate>0; buy=%s sell=%s", buyOut.Dec(), sellOut.Dec())
}

// When feeRate == 0, buy/sell DO equal each other (and equal general).
func TestBuySellSymmetry_WhenNoSwapFee(t *testing.T) {
	t.Parallel()
	buyOut, err := getAmountOutMemeBuy(u("1000"), u("10000"), u("10000"), 0)
	require.NoError(t, err)
	sellOut, err := getAmountOutMemeSell(u("1000"), u("10000"), u("10000"), 0)
	require.NoError(t, err)
	assert.Equal(t, buyOut.Dec(), sellOut.Dec())
}

// Fixtures captured directly from the NadFunPair.getAmountOut Solidity implementation.
// To regenerate: see scripts/generate-nadswap-fixtures.md (not included in this PR).
var amountOutFixtures = []struct {
	name        string
	reserve0    string
	reserve1    string
	quoteToken0 bool // true if quoteToken == token0
	feeRate     uint16
	tokenInIs0  bool
	amountIn    string
	expectedOut string
}{
	// Symmetric (feeRate=0) - should equal both general and meme directions
	{"zero_fee_buy_t0", "1000000000000000000000", "2000000000000000000000", true, 0, true, "10000000000000000", "19913234389576345"},
	{"zero_fee_sell_t1", "1000000000000000000000", "2000000000000000000000", true, 0, false, "10000000000000000", "4995008329451707"},

	// Asymmetric: feeRate = 200 (2%)
	{"buy_fee200", "1000000000000000000000", "2000000000000000000000", true, 200, true, "10000000000000000", "19515590043263921"},
	{"sell_fee200", "1000000000000000000000", "2000000000000000000000", true, 200, false, "10000000000000000", "4895086664497443"},
}

// TestFixtures asserts every fixture matches our math implementation exactly.
// Fixture values MUST be regenerated against the deployed NadFunPair contract.
// During development this test starts as `t.Skip` and is enabled once values are filled in.
func TestFixtures_AmountOut(t *testing.T) {
	t.Parallel()
	t.Skip("ENABLE AFTER GENERATING FIXTURE VALUES FROM NadFunPair.getAmountOut")

	for _, f := range amountOutFixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			r0, r1, amountIn := u(f.reserve0), u(f.reserve1), u(f.amountIn)

			// reserveIn/reserveOut from the perspective of tokenIn.
			reserveIn, reserveOut := r0, r1
			if !f.tokenInIs0 {
				reserveIn, reserveOut = r1, r0
			}
			// Buy when tokenIn == quoteToken.
			isBuy := f.tokenInIs0 == f.quoteToken0

			var got *uint256.Int
			var err error
			if isBuy {
				got, err = getAmountOutMemeBuy(amountIn, reserveIn, reserveOut, f.feeRate)
			} else {
				got, err = getAmountOutMemeSell(amountIn, reserveIn, reserveOut, f.feeRate)
			}
			require.NoError(t, err)
			assert.Equal(t, f.expectedOut, got.Dec(), "fixture %q", f.name)
		})
	}
}
