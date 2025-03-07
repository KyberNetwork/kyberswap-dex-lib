package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextSqrtRatioFromAmount0AddPriceGoesDown(t *testing.T) {
	res, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		big.NewInt(1_000_000),
		big.NewInt(1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("339942424496442021441932674757011200256")))
}

func TestNextSqrtRatioFromAmount0ExactOutOverflow(t *testing.T) {
	_, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		One,
		IntFromString("-100_000_000_000_000"),
	)
	require.Error(t, err)
}

func TestNextSqrtRatioFromAmount0ExactInCantUnderflow(t *testing.T) {
	res, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		One,
		IntFromString("100_000_000_000_000"),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("3402823669209350606397054")))
}

func TestNextSqrtRatioFromAmount0SubPriceGoesUp(t *testing.T) {
	res, err := nextSqrtRatioFromAmount0(
		TwoPow128,
		IntFromString("100_000_000_000"),
		big.NewInt(-1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("340282370323762166700996274441730955874")))
}

func TestNextSqrtRatioFromAmount1AddPriceGoesUp(t *testing.T) {
	res, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		IntFromString("1_000_000"),
		big.NewInt(1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("340622649287859401926837982039199979667")))
}

func TestNextSqrtRatioFromAmount1ExactOutOverflow(t *testing.T) {
	_, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		One,
		IntFromString("-100_000_000_000_000"),
	)
	require.Error(t, err)
}

func TestNextSqrtRatioFromAmount1ExactInCantUnderflow(t *testing.T) {
	res, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		One,
		IntFromString("100_000_000_000_000"),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("34028236692094186628704381681640284520207431768211456")))
}

func TestNextSqrtRatioFromAmount1SubPriceGoesDown(t *testing.T) {
	res, err := nextSqrtRatioFromAmount1(
		TwoPow128,
		IntFromString("100_000_000_000"),
		big.NewInt(-1000),
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("340282363518114794253989972798022137138")))
}
