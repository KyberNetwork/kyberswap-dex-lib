package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestNextSqrtRatioFromAmount0AddPriceGoesDown(t *testing.T) {
	t.Parallel()
	res, err := nextSqrtRatioFromAmount0(
		big256.U2Pow128,
		uint256.NewInt(1_000_000),
		uint256.NewInt(1000),
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("339942424496442021441932674757011200256"), res)
}

func TestNextSqrtRatioFromAmount0ExactOutOverflow(t *testing.T) {
	t.Parallel()
	_, err := nextSqrtRatioFromAmount0(
		big256.U2Pow128,
		big256.U1,
		new(uint256.Int).Neg(uint256.NewInt(1e14)),
	)
	require.Error(t, err)
}

func TestNextSqrtRatioFromAmount0ExactInCantUnderflow(t *testing.T) {
	t.Parallel()
	res, err := nextSqrtRatioFromAmount0(
		big256.U2Pow128,
		big256.U1,
		big256.New("100000000000000"),
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("3402823669209350606397054"), res)
}

func TestNextSqrtRatioFromAmount0SubPriceGoesUp(t *testing.T) {
	t.Parallel()
	res, err := nextSqrtRatioFromAmount0(
		big256.U2Pow128,
		big256.New("100000000000"),
		new(uint256.Int).Neg(uint256.NewInt(1000)),
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("340282370323762166700996274441730955874"), res)
}

func TestNextSqrtRatioFromAmount1AddPriceGoesUp(t *testing.T) {
	t.Parallel()
	res, err := nextSqrtRatioFromAmount1(
		big256.U2Pow128,
		big256.New("1000000"),
		uint256.NewInt(1000),
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("340622649287859401926837982039199979667"), res)
}

func TestNextSqrtRatioFromAmount1ExactOutOverflow(t *testing.T) {
	t.Parallel()
	_, err := nextSqrtRatioFromAmount1(
		big256.U2Pow128,
		big256.U1,
		new(uint256.Int).Neg(uint256.NewInt(1e14)),
	)
	require.Error(t, err)
}

func TestNextSqrtRatioFromAmount1ExactInCantUnderflow(t *testing.T) {
	t.Parallel()
	res, err := nextSqrtRatioFromAmount1(
		big256.U2Pow128,
		big256.U1,
		big256.New("100000000000000"),
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("34028236692094186628704381681640284520207431768211456"), res)
}

func TestNextSqrtRatioFromAmount1SubPriceGoesDown(t *testing.T) {
	t.Parallel()
	res, err := nextSqrtRatioFromAmount1(
		big256.U2Pow128,
		big256.New("100000000000"),
		new(uint256.Int).Neg(uint256.NewInt(1000)),
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("340282363518114794253989972798022137138"), res)
}

func TestFloatSqrtRatioToFixed(t *testing.T) {
	t.Parallel()
	require.Equal(t, big256.New("684473135231248274430278569558016"),
		FloatSqrtRatioToFixed(big256.New("19807080470146244316807077133")))
	require.Equal(t, big256.New("1135857851277201550342163271201301987328"),
		FloatSqrtRatioToFixed(big256.New("39614081272525913171640211645")))
	require.Equal(t, big256.New("860225444998530776275902070124017876992"),
		FloatSqrtRatioToFixed(big256.New("39614081268790397500455936071")))

}
