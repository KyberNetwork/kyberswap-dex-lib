package umbraedlmm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestNormalizedPriceFromId(t *testing.T) {
	// Centre bin is exactly 1.0 in 1e18.
	require.Equal(t, e18, getNormalizedPriceFromId(activeBinID, 25))
	// One bin up at binStep 25: 1e18*10025/10000.
	require.Equal(t, uint256.NewInt(1002500000000000000), getNormalizedPriceFromId(activeBinID+1, 25))
	// One bin down: floor(1e18*10000/10025) — the deployed 1e18 exponentiation is exact here.
	require.Equal(t, uint256.NewInt(997506234413965087), getNormalizedPriceFromId(activeBinID-1, 25))
}

func TestIsNormalizedPriceRepresentable(t *testing.T) {
	require.True(t, isNormalizedPriceRepresentable(e18))
	require.False(t, isNormalizedPriceRepresentable(uint256.NewInt(0)))
	require.False(t, isNormalizedPriceRepresentable(new(uint256.Int).Set(uMaxU)))
}

func TestMulDivFloorCeil(t *testing.T) {
	// floor(7*3/2) = 10, ceil = 11 — split-formula path: 7*(3/2) + 7*(3%2)/2 = 7 + 3 = 10.
	f, err := mulDivFloor(uint256.NewInt(7), uint256.NewInt(3), uint256.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(10), f)
	c, err := mulDivCeil(uint256.NewInt(7), uint256.NewInt(3), uint256.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(11), c)
	// Exact division: floor == ceil.
	f, err = mulDivFloor(uint256.NewInt(6), uint256.NewInt(4), uint256.NewInt(3))
	require.NoError(t, err)
	c, err = mulDivCeil(uint256.NewInt(6), uint256.NewInt(4), uint256.NewInt(3))
	require.NoError(t, err)
	require.Equal(t, f, c)
	// The intermediates are NOT 512-bit: a*(b/d) overflowing must error like the on-chain revert.
	_, err = mulDivFloor(uMaxU, uMaxU, uint256.NewInt(1))
	require.ErrorIs(t, err, ErrMathOverflow)
}

func TestCalculateDynamicFee(t *testing.T) {
	// vol=0 -> just the base factor.
	require.Equal(t, uint256.NewInt(30), calculateDynamicFee(30, 4000, 0, 25, 100))
	// vol=100, binStep=25, control=4000: (100*25)^2 * 4000 / 1e10 = 2 -> 32 bps.
	require.Equal(t, uint256.NewInt(32), calculateDynamicFee(30, 4000, 100, 25, 100))
	// variableFeeControl=0 disables the variable term.
	require.Equal(t, uint256.NewInt(30), calculateDynamicFee(30, 0, 9999, 25, 100))
	// V2 #147: the cap clamp is UNCONDITIONAL — a zero cap pins the variable part to zero.
	require.Equal(t, uint256.NewInt(30), calculateDynamicFee(30, 4000, 100, 25, 0))
	// variableFeeCap caps the variable term: vol=100 gives variableFee 2, cap at 1 -> 31 bps.
	require.Equal(t, uint256.NewInt(31), calculateDynamicFee(30, 4000, 100, 25, 1))
	// Capped at MAX_FEE (500) when the capped variable fee still exceeds it.
	require.Equal(t, uMaxFee, calculateDynamicFee(30, 4000, 35000, 25, 65535))
}

func TestGetFeeAmountFrom(t *testing.T) {
	// V2 ceils: 1e18 at 30 bps = ceil(1e18*30/10030) = 2991026919242274 (V1 floored to ...273).
	fee, err := getFeeAmountFrom(e18, uint256.NewInt(30))
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(2991026919242274), fee)
	// The fee can never consume the whole amount: ceil(1*30/10030)=1 >= 1 clamps to amount-1 = 0.
	fee, err = getFeeAmountFrom(uint256.NewInt(1), uint256.NewInt(30))
	require.NoError(t, err)
	require.True(t, fee.IsZero())
	// Zero fee rate charges nothing.
	fee, err = getFeeAmountFrom(e18, uint256.NewInt(0))
	require.NoError(t, err)
	require.True(t, fee.IsZero())
}

func TestVolatilityDecay(t *testing.T) {
	// Inside the filter window the accumulator holds constant.
	require.Equal(t, uint64(1000), getDecayedVolatility(1000, 100, 30, 600, 120))
	// Past the filter window it decays linearly over decayPeriod: delta=300 of 600 -> half.
	require.Equal(t, uint64(500), getDecayedVolatility(1000, 100, 30, 600, 400))
	// Past decayPeriod it is zero.
	require.Equal(t, uint64(0), getDecayedVolatility(1000, 100, 30, 600, 701))
	// Non-zero floor of 1 while decaying.
	require.Equal(t, uint64(1), getDecayedVolatility(1, 100, 30, 600, 400))
}
