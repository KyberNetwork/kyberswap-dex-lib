package quoting

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/math"
	"github.com/stretchr/testify/require"
)

func TestNearestInitializedTickIndex(t *testing.T) {
	t.Run("no ticks", func(t *testing.T) {
		require.Equal(t, InvalidTickIndex, NearestInitializedTickIndex([]Tick{}, 0))
	})

	t.Run("index zero tick less than", func(t *testing.T) {
		require.Equal(t, 0, NearestInitializedTickIndex([]Tick{
			{
				Number:         -1,
				LiquidityDelta: math.One,
			},
		}, 0))
	})

	t.Run("index zero tick equal to", func(t *testing.T) {
		require.Equal(t, 0, NearestInitializedTickIndex([]Tick{
			{
				Number:         0,
				LiquidityDelta: math.One,
			},
		}, 0))
	})

	t.Run("index zero tick greater than", func(t *testing.T) {
		require.Equal(t, InvalidTickIndex, NearestInitializedTickIndex([]Tick{
			{
				Number:         1,
				LiquidityDelta: math.One,
			},
		}, 0))
	})

	t.Run("many ticks", func(t *testing.T) {
		ticks := []Tick{
			{
				Number:         -100,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         -5,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         -4,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         18,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         23,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         50,
				LiquidityDelta: new(big.Int),
			},
		}

		require.Equal(t, InvalidTickIndex, NearestInitializedTickIndex(ticks, -101))
		require.Equal(t, 0, NearestInitializedTickIndex(ticks, -100))
		require.Equal(t, 0, NearestInitializedTickIndex(ticks, -99))
		require.Equal(t, 0, NearestInitializedTickIndex(ticks, -6))
		require.Equal(t, 1, NearestInitializedTickIndex(ticks, -5))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, -4))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, -3))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, 0))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, 17))
		require.Equal(t, 3, NearestInitializedTickIndex(ticks, 18))
		require.Equal(t, 3, NearestInitializedTickIndex(ticks, 19))
		require.Equal(t, 3, NearestInitializedTickIndex(ticks, 22))
		require.Equal(t, 4, NearestInitializedTickIndex(ticks, 23))
		require.Equal(t, 4, NearestInitializedTickIndex(ticks, 24))
		require.Equal(t, 4, NearestInitializedTickIndex(ticks, 49))
		require.Equal(t, 5, NearestInitializedTickIndex(ticks, 50))
		require.Equal(t, 5, NearestInitializedTickIndex(ticks, 51))
	})
}
