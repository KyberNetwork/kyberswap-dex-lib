package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestZeroAmountToken0(t *testing.T) {
	t.Parallel()
	sqrtRatio := big256.U2Pow128

	res, err := ComputeStep(
		sqrtRatio,
		uint256.NewInt(100_000),
		MinSqrtRatio,
		new(uint256.Int),
		false,
		0,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Sign())
	require.Zero(t, res.ConsumedAmount.Sign())
	require.Zero(t, res.FeeAmount.Sign())
	require.Equal(t, sqrtRatio, res.SqrtRatioNext)
}

func TestZeroAmountToken1(t *testing.T) {
	t.Parallel()
	sqrtRatio := big256.U2Pow128

	res, err := ComputeStep(
		sqrtRatio,
		uint256.NewInt(100_000),
		MinSqrtRatio,
		new(uint256.Int),
		true,
		0,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Sign())
	require.Zero(t, res.ConsumedAmount.Sign())
	require.Zero(t, res.FeeAmount.Sign())
	require.Equal(t, sqrtRatio, res.SqrtRatioNext)
}

func TestSwapRatioEqualLimitToken1(t *testing.T) {
	t.Parallel()
	sqrtRatio := big256.U2Pow128

	res, err := ComputeStep(
		sqrtRatio,
		uint256.NewInt(100_000),
		sqrtRatio,
		uint256.NewInt(10_000),
		true,
		0,
	)
	require.NoError(t, err)

	require.Zero(t, res.CalculatedAmount.Sign())
	require.Zero(t, res.ConsumedAmount.Sign())
	require.Zero(t, res.FeeAmount.Sign())
	require.Equal(t, sqrtRatio, res.SqrtRatioNext)
}

func TestMaxLimitToken0Input(t *testing.T) {
	t.Parallel()
	sqrtRatio := big256.U2Pow128

	res, err := ComputeStep(
		sqrtRatio,
		uint256.NewInt(100_000),
		MinSqrtRatio,
		uint256.NewInt(10_000),
		false,
		1<<63,
	)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(4_761), res.CalculatedAmount)
	require.Equal(t, uint256.NewInt(10_000), res.ConsumedAmount)
	require.Equal(t, uint256.NewInt(5_000), res.FeeAmount)
	require.Equal(t, big256.New("324078444686608060441309149948106768384"), res.SqrtRatioNext)
}

func TestMaxLimitToken1Input(t *testing.T) {
	t.Parallel()
	res, err := ComputeStep(
		big256.U2Pow128,
		uint256.NewInt(100_000),
		MaxSqrtRatio,
		uint256.NewInt(10_000),
		true,
		1<<63,
	)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(4_761), res.CalculatedAmount)
	require.Equal(t, uint256.NewInt(10_000), res.ConsumedAmount)
	require.Equal(t, uint256.NewInt(5_000), res.FeeAmount)
	require.Equal(t, big256.New("357296485266985386621785942544388980736"), res.SqrtRatioNext)
}

func TestMaxLimitToken0Output(t *testing.T) {
	t.Parallel()
	res, err := ComputeStep(
		big256.U2Pow128,
		uint256.NewInt(100_000),
		MaxSqrtRatio,
		new(uint256.Int).Neg(uint256.NewInt(10_000)),
		false,
		1<<63,
	)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(22_224), res.CalculatedAmount)
	require.Equal(t, new(uint256.Int).Neg(uint256.NewInt(10_000)), res.ConsumedAmount)
	require.Equal(t, uint256.NewInt(11_112), res.FeeAmount)
	require.Equal(t, big256.New("378091518801042737222520106199097016320"), res.SqrtRatioNext)
}

func TestMaxLimitToken1Output(t *testing.T) {
	t.Parallel()
	res, err := ComputeStep(
		big256.U2Pow128,
		uint256.NewInt(100_000),
		MinSqrtRatio,
		new(uint256.Int).Neg(uint256.NewInt(10_000)),
		true,
		1<<63,
	)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(22_224), res.CalculatedAmount)
	require.Equal(t, new(uint256.Int).Neg(uint256.NewInt(10_000)), res.ConsumedAmount)
	require.Equal(t, uint256.NewInt(11_112), res.FeeAmount)
	require.Equal(t, big256.New("306254130228844617117037146678283468800"), res.SqrtRatioNext)
}

func TestLimitedToken0Output(t *testing.T) {
	t.Parallel()
	sqrtRatioLimit := big256.New("359186942860990600322450974511310889870")

	res, err := ComputeStep(
		big256.U2Pow128,
		uint256.NewInt(100_000),
		sqrtRatioLimit,
		new(uint256.Int).Neg(uint256.NewInt(10_000)),
		false,
		1<<63,
	)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(11_112), res.CalculatedAmount)
	require.Equal(t, new(uint256.Int).Neg(uint256.NewInt(5_263)), res.ConsumedAmount)
	require.Equal(t, uint256.NewInt(5_556), res.FeeAmount)
	require.Equal(t, sqrtRatioLimit, res.SqrtRatioNext)
}

func TestLimitedToken1Output(t *testing.T) {
	t.Parallel()
	sqrtRatioLimit := big256.New("323268248574891540290205877060179800883")

	res, err := ComputeStep(
		big256.U2Pow128,
		uint256.NewInt(100_000),
		sqrtRatioLimit,
		new(uint256.Int).Neg(uint256.NewInt(10_000)),
		true,
		1<<63,
	)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(10_528), res.CalculatedAmount)
	require.Equal(t, new(uint256.Int).Neg(uint256.NewInt(5_000)), res.ConsumedAmount)
	require.Equal(t, uint256.NewInt(5_264), res.FeeAmount)
	require.Equal(t, sqrtRatioLimit, res.SqrtRatioNext)
}

func TestRoundSqrtRatio(t *testing.T) {
	t.Parallel()

	for _, ratio := range []*uint256.Int{MinSqrtRatio, MaxSqrtRatio, ToSqrtRatio(-1_000_000), ToSqrtRatio(12_345_678)} {
		require.True(t, IsValidSqrtRatio(ratio))
		require.Equal(t, ratio, RoundSqrtRatio(ratio, false))
		require.Equal(t, ratio, RoundSqrtRatio(ratio, true))
		above := new(uint256.Int).AddUint64(ratio, 1)
		require.False(t, IsValidSqrtRatio(above))
		require.Equal(t, ratio, RoundSqrtRatio(above, false))
		require.True(t, RoundSqrtRatio(above, true).Gt(above))
	}
	// rounding up into the next region
	require.Equal(t, big256.U2Pow96, RoundSqrtRatio(new(uint256.Int).SubUint64(big256.U2Pow96, 1), true))
	require.Equal(t, new(uint256.Int).SetAllOne(), RoundSqrtRatio(new(uint256.Int).SetAllOne(), true))
}
