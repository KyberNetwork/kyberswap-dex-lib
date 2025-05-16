package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestZeroAmountToken0(t *testing.T) {
	t.Parallel()
	sqrtRatio := TwoPow128

	res, err := ComputeStep(
		sqrtRatio,
		big.NewInt(100_000),
		MinSqrtRatio,
		new(big.Int),
		false,
		0,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Sign())
	require.Zero(t, res.ConsumedAmount.Sign())
	require.Zero(t, res.FeeAmount.Sign())
	require.Zero(t, res.SqrtRatioNext.Cmp(sqrtRatio))
}

func TestZeroAmountToken1(t *testing.T) {
	t.Parallel()
	sqrtRatio := TwoPow128

	res, err := ComputeStep(
		sqrtRatio,
		big.NewInt(100_000),
		MinSqrtRatio,
		new(big.Int),
		true,
		0,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Sign())
	require.Zero(t, res.ConsumedAmount.Sign())
	require.Zero(t, res.FeeAmount.Sign())
	require.Zero(t, res.SqrtRatioNext.Cmp(sqrtRatio))
}

func TestSwapRatioEqualLimitToken1(t *testing.T) {
	t.Parallel()
	sqrtRatio := TwoPow128

	res, err := ComputeStep(
		sqrtRatio,
		big.NewInt(100_000),
		sqrtRatio,
		big.NewInt(10_000),
		true,
		0,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Sign())
	require.Zero(t, res.ConsumedAmount.Sign())
	require.Zero(t, res.FeeAmount.Sign())
	require.Zero(t, res.SqrtRatioNext.Cmp(sqrtRatio))
}

func TestMaxLimitToken0Input(t *testing.T) {
	t.Parallel()
	sqrtRatio := TwoPow128

	res, err := ComputeStep(
		sqrtRatio,
		big.NewInt(100_000),
		MinSqrtRatio,
		big.NewInt(10_000),
		false,
		1<<63,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Cmp(big.NewInt(4_761)))
	require.Zero(t, res.ConsumedAmount.Cmp(big.NewInt(10_000)))
	require.Zero(t, res.FeeAmount.Cmp(big.NewInt(5_000)))
	require.Zero(t, res.SqrtRatioNext.Cmp(bignum.NewBig("324078444686608060441309149935017344244")))
}

func TestMaxLimitToken1Input(t *testing.T) {
	t.Parallel()
	res, err := ComputeStep(
		TwoPow128,
		big.NewInt(100_000),
		MaxSqrtRatio,
		big.NewInt(10_000),
		true,
		1<<63,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Cmp(big.NewInt(4_761)))
	require.Zero(t, res.ConsumedAmount.Cmp(big.NewInt(10_000)))
	require.Zero(t, res.FeeAmount.Cmp(big.NewInt(5_000)))
	require.Zero(t, res.SqrtRatioNext.Cmp(bignum.NewBig("357296485266985386636543337803356622028")))
}

func TestMaxLimitToken0Output(t *testing.T) {
	t.Parallel()
	res, err := ComputeStep(
		TwoPow128,
		big.NewInt(100_000),
		MaxSqrtRatio,
		big.NewInt(-10_000),
		false,
		1<<63,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Cmp(big.NewInt(22_224)))
	require.Zero(t, res.ConsumedAmount.Cmp(big.NewInt(-10_000)))
	require.Zero(t, res.FeeAmount.Cmp(big.NewInt(11_112)))
	require.Zero(t, res.SqrtRatioNext.Cmp(bignum.NewBig("378091518801042737181527341590853568285")))
}

func TestMaxLimitToken1Output(t *testing.T) {
	t.Parallel()
	res, err := ComputeStep(
		TwoPow128,
		big.NewInt(100_000),
		MinSqrtRatio,
		big.NewInt(-10_000),
		true,
		1<<63,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Cmp(big.NewInt(22_224)))
	require.Zero(t, res.ConsumedAmount.Cmp(big.NewInt(-10_000)))
	require.Zero(t, res.FeeAmount.Cmp(big.NewInt(11_112)))
	require.Zero(t, res.SqrtRatioNext.Cmp(bignum.NewBig("306254130228844617117037146688591390310")))
}

func TestLimitedToken0Output(t *testing.T) {
	t.Parallel()
	sqrtRatioLimit := bignum.NewBig("359186942860990600322450974511310889870")

	res, err := ComputeStep(
		TwoPow128,
		big.NewInt(100_000),
		sqrtRatioLimit,
		big.NewInt(-10_000),
		false,
		1<<63,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Cmp(big.NewInt(11_112)))
	require.Zero(t, res.ConsumedAmount.Cmp(big.NewInt(-5_263)))
	require.Zero(t, res.FeeAmount.Cmp(big.NewInt(5_556)))
	require.Zero(t, res.SqrtRatioNext.Cmp(sqrtRatioLimit))
}

func TestLimitedToken1Output(t *testing.T) {
	t.Parallel()
	sqrtRatioLimit := bignum.NewBig("323268248574891540290205877060179800883")

	res, err := ComputeStep(
		TwoPow128,
		big.NewInt(100_000),
		sqrtRatioLimit,
		big.NewInt(-10_000),
		true,
		1<<63,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Cmp(big.NewInt(10_528)))
	require.Zero(t, res.ConsumedAmount.Cmp(big.NewInt(-5_000)))
	require.Zero(t, res.FeeAmount.Cmp(big.NewInt(5_264)))
	require.Zero(t, res.SqrtRatioNext.Cmp(sqrtRatioLimit))
}
