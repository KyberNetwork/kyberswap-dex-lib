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

func TestGetPriceFromId_DecimalScaling(t *testing.T) {
	// 18-decimal quote: price equals the normalized price.
	p, err := getPriceFromId(activeBinID, 25, 18)
	require.NoError(t, err)
	require.Equal(t, e18, p)

	// 6-decimal quote (e.g. USDC): price = normalized / 10^12 = 10^6 at the centre bin.
	p6, err := getPriceFromId(activeBinID, 25, 6)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(1000000), p6)
}

func TestCalculateDynamicFee(t *testing.T) {
	// vol=0 -> just the base factor (varFeeCap 0 = no cap).
	require.Equal(t, uint256.NewInt(30), calculateDynamicFee(30, 4000, uint256.NewInt(0), 25, 0))
	// vol=100, binStep=25, control=4000: (100*25)^2 * 4000 / 1e10 = 2 -> 32 bps.
	require.Equal(t, uint256.NewInt(32), calculateDynamicFee(30, 4000, uint256.NewInt(100), 25, 0))
	// variableFeeControl=0 disables the variable term.
	require.Equal(t, uint256.NewInt(30), calculateDynamicFee(30, 0, uint256.NewInt(9999), 25, 0))
	// Capped at MAX_FEE (500) when uncapped variable fee is huge.
	require.Equal(t, uMaxFee, calculateDynamicFee(30, 4000, uint256.NewInt(35000), 25, 0))
	// variableFeeCap caps the variable term: vol=100 gives variableFee 2, cap at 1 -> 31 bps.
	require.Equal(t, uint256.NewInt(31), calculateDynamicFee(30, 4000, uint256.NewInt(100), 25, 1))
}

func TestGetFeeAmountFrom(t *testing.T) {
	// 1e18 at 30 bps fee = floor(1e18*30/10030) = floor(3e19/10030).
	require.Equal(t, uint256.NewInt(2991026919242273), getFeeAmountFrom(e18, uint256.NewInt(30)))
}
