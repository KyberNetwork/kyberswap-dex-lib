package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestNextSqrtRatioFromAmount0AddPriceGoesDown(t *testing.T) {
	res, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		big.NewInt(1_000_000),
		big.NewInt(1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("339942424496442021441932674757011200256")))
}

func TestNextSqrtRatioFromAmount0ExactOutOverflow(t *testing.T) {
	_, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		bignum.One,
		bignum.NewBig("-100_000_000_000_000"),
	)
	require.Error(t, err)
}

func TestNextSqrtRatioFromAmount0ExactInCantUnderflow(t *testing.T) {
	res, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		bignum.One,
		bignum.NewBig("100_000_000_000_000"),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("3402823669209350606397054")))
}

func TestNextSqrtRatioFromAmount0SubPriceGoesUp(t *testing.T) {
	res, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		bignum.NewBig("100_000_000_000"),
		big.NewInt(-1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("340282370323762166700996274441730955874")))
}

func TestNextSqrtRatioFromAmount1AddPriceGoesUp(t *testing.T) {
	res, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		bignum.NewBig("1_000_000"),
		big.NewInt(1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("340622649287859401926837982039199979667")))
}

func TestNextSqrtRatioFromAmount1ExactOutOverflow(t *testing.T) {
	_, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		bignum.One,
		bignum.NewBig("-100_000_000_000_000"),
	)
	require.Error(t, err)
}

func TestNextSqrtRatioFromAmount1ExactInCantUnderflow(t *testing.T) {
	res, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		bignum.One,
		bignum.NewBig("100_000_000_000_000"),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("34028236692094186628704381681640284520207431768211456")))
}

func TestNextSqrtRatioFromAmount1SubPriceGoesDown(t *testing.T) {
	res, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		bignum.NewBig("100_000_000_000"),
		big.NewInt(-1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("340282363518114794253989972798022137138")))
}

func TestFloatSqrtRatioToFixed(t *testing.T) {
	require.Zero(t, FloatSqrtRatioToFixed(bignum.NewBig("19807080470146244316807077133")).
		Cmp(bignum.NewBig("684473135231248274430278569558016")))
	require.Zero(t, FloatSqrtRatioToFixed(bignum.NewBig("39614081272525913171640211645")).
		Cmp(bignum.NewBig("1135857851277201550342163271201301987328")))
	require.Zero(t, FloatSqrtRatioToFixed(bignum.NewBig("39614081268790397500455936071")).
		Cmp(bignum.NewBig("860225444998530776275902070124017876992")))

}
