package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestToSqrtRatio(t *testing.T) {
	t.Parallel()

	t.Run("examples", func(t *testing.T) {
		require.Equal(t, big256.New("561030636129153856579134353873645338624"), ToSqrtRatio(1_000_000))
		require.Equal(t, big256.New("50502254805927926084423855178401471004672"), ToSqrtRatio(10_000_000))
		require.Equal(t, big256.New("206391740095027370700312310528859963392"), ToSqrtRatio(-1_000_000))
		require.Equal(t, big256.New("2292810285051363400276741630355046400"), ToSqrtRatio(-10_000_000))
	})

	t.Run("min tick", func(t *testing.T) {
		require.Equal(t, MinSqrtRatio, ToSqrtRatio(MinTick))
	})

	t.Run("max tick", func(t *testing.T) {
		require.Equal(t, MaxSqrtRatio, ToSqrtRatio(MaxTick))
	})
}

func TestApproximateNumberOfTickSpacingsCrossed(t *testing.T) {
	t.Parallel()

	t.Run("doubling", func(t *testing.T) {
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(MinSqrtRatio, new(uint256.Int).Mul(big256.U2, MinSqrtRatio), 0), uint32(0))

		// 2x sqrt ratio increase ~= 4x price increase
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(big256.U2Pow128, new(uint256.Int).Mul(big256.U2,
			big256.U2Pow128), 1), uint32(1386295))
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(MinSqrtRatio, new(uint256.Int).Mul(big256.U2, MinSqrtRatio), 1), uint32(1386295))
		require.Equal(t, ApproximateNumberOfTickSpacingsCrossed(MaxSqrtRatio, new(uint256.Int).Div(MaxSqrtRatio, big256.U2), 1), uint32(1386295))
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
	base := new(uint256.Int).Lsh(uint256.NewInt(1), 128)
	double := new(uint256.Int).Lsh(base, 1)

	require.Equal(t, uint32(1386295), ApproximateNumberOfTickSpacingsCrossed(base, double, 1))
	require.Equal(t, int32(0), ApproximateSqrtRatioToTick(ToSqrtRatio(0)))
}
