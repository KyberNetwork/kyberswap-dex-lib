package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestToSqrtRatio(t *testing.T) {
	t.Parallel()

	t.Run("examples", func(t *testing.T) {
		require.Zero(t, ToSqrtRatio(1_000_000).Cmp(bignum.NewBig("561030636129153856579134353873645338624")))
		require.Zero(t, ToSqrtRatio(10_000_000).Cmp(bignum.NewBig("50502254805927926084423855178401471004672")))
		require.Zero(t, ToSqrtRatio(-1_000_000).Cmp(bignum.NewBig("206391740095027370700312310528859963392")))
		require.Zero(t, ToSqrtRatio(-10_000_000).Cmp(bignum.NewBig("2292810285051363400276741630355046400")))
	})

	t.Run("min tick", func(t *testing.T) {
		require.Zero(t, ToSqrtRatio(MinTick).Cmp(MinSqrtRatio))
	})

	t.Run("max tick", func(t *testing.T) {
		require.Zero(t, ToSqrtRatio(MaxTick).Cmp(MaxSqrtRatio))
	})
}

func TestApproximateNumberOfTickSpacingsCrossed(t *testing.T) {
	t.Parallel()

	t.Run("doubling", func(t *testing.T) {
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(MinSqrtRatio, new(big.Int).Mul(bignum.Two, MinSqrtRatio), 0), uint32(0))

		// 2x sqrt ratio increase ~= 4x price increase
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(TwoPow128, new(big.Int).Mul(bignum.Two, TwoPow128), 1), uint32(1386295))
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(MinSqrtRatio, new(big.Int).Mul(bignum.Two, MinSqrtRatio), 1), uint32(1386295))
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(MaxSqrtRatio, new(big.Int).Div(MaxSqrtRatio, bignum.Two), 1), uint32(1386295))
	})

	t.Run("doubling big tick spacing", func(t *testing.T) {

	})
}

func TestApproximateSqrtRatioToTick(t *testing.T) {
	t.Parallel()

	t.Run("examples", func(t *testing.T) {
		require.Equal(t, ApproximateSqrtRatioToTick(ToSqrtRatio(0)), int32(0))
		require.Equal(t, ApproximateSqrtRatioToTick(ToSqrtRatio(1000000)), int32(1000000))
		require.Equal(t, ApproximateSqrtRatioToTick(ToSqrtRatio(10000000)), int32(10000000))
		require.Equal(t, ApproximateSqrtRatioToTick(ToSqrtRatio(-1000000)), int32(-1000000))
		require.Equal(t, ApproximateSqrtRatioToTick(ToSqrtRatio(-10000000)), int32(-10000000))
	})

	t.Run("min tick", func(t *testing.T) {
		require.Equal(t, ApproximateSqrtRatioToTick(ToSqrtRatio(MinTick)), MinTick)
	})

	t.Run("max tick", func(t *testing.T) {
		require.Equal(t, ApproximateSqrtRatioToTick(ToSqrtRatio(MaxTick)), MaxTick)
	})
}

func TestAbc(t *testing.T) {
	t.Parallel()
	base := new(big.Int).Lsh(big.NewInt(1), 128)
	double := new(big.Int).Lsh(base, 1)

	require.Equal(t, uint32(1386295), ApproximateNumberOfTickSpacingsCrossed(base, double, 1))
	require.Equal(t, int32(0), ApproximateSqrtRatioToTick(ToSqrtRatio(0)))
}
