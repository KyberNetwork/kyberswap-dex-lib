package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelta(t *testing.T) {
	t.Run("amount0", func(t *testing.T) {
		t.Run("priceDown", func(t *testing.T) {
			result, err := amount0Delta(
				IntFromString("339942424496442021441932674757011200255"),
				TwoPow128,
				big.NewInt(1_000_000),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, big.NewInt(1_000))
		})

		t.Run("priceDownReverse", func(t *testing.T) {
			result, err := amount0Delta(
				TwoPow128,
				IntFromString("339942424496442021441932674757011200255"),
				big.NewInt(1_000_000),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, big.NewInt(1_000))
		})

		t.Run("priceExampleDown", func(t *testing.T) {
			result, err := amount0Delta(
				TwoPow128,
				new(big.Int).Add(IntFromString("34028236692093846346337460743176821145"), TwoPow128),
				IntFromString("1_000_000_000_000_000_000"),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, IntFromString("90_909_090_909_090_909"))
		})

		t.Run("priceExampleUp", func(t *testing.T) {
			result, err := amount0Delta(
				TwoPow128,
				new(big.Int).Add(IntFromString("34028236692093846346337460743176821145"), TwoPow128),
				IntFromString("1_000_000_000_000_000_000"),
				true,
			)
			require.NoError(t, err)
			require.Equal(t, result, IntFromString("90_909_090_909_090_910"))
		})
	})

	t.Run("amount1", func(t *testing.T) {
		t.Run("priceDown", func(t *testing.T) {
			result, err := amount1Delta(
				IntFromString("339942424496442021441932674757011200255"),
				TwoPow128,
				big.NewInt(1_000_000),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, big.NewInt(999))
		})

		t.Run("priceDownReverse", func(t *testing.T) {
			result, err := amount1Delta(
				TwoPow128,
				IntFromString("339942424496442021441932674757011200255"),
				big.NewInt(1_000_000),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, big.NewInt(999))
		})

		t.Run("priceUp", func(t *testing.T) {
			result, err := amount1Delta(
				new(big.Int).Add(IntFromString("340622989910849312776150758189957120"), TwoPow128),
				TwoPow128,
				big.NewInt(1_000_000),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, big.NewInt(1001))
		})

		t.Run("priceUpReverse", func(t *testing.T) {
			result, err := amount1Delta(
				TwoPow128,
				IntFromString("339942424496442021441932674757011200255"),
				big.NewInt(1_000_000),
				true,
			)
			require.NoError(t, err)
			require.Equal(t, result, big.NewInt(1000))
		})

		t.Run("priceExampleDown", func(t *testing.T) {
			result, err := amount1Delta(
				TwoPow128,
				IntFromString("309347606291762239512158734028880192232"),
				IntFromString("1_000_000_000_000_000_000"),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, IntFromString("90_909_090_909_090_909"))
		})

		t.Run("priceExampleUp", func(t *testing.T) {
			result, err := amount1Delta(
				TwoPow128,
				IntFromString("309347606291762239512158734028880192232"),
				IntFromString("1_000_000_000_000_000_000"),
				true,
			)
			require.NoError(t, err)
			require.Equal(t, result, IntFromString("90_909_090_909_090_910"))
		})

		t.Run("noOverflowHalfPriceRange", func(t *testing.T) {
			result, err := amount1Delta(
				TwoPow128,
				MaxSqrtRatio,
				IntFromString("0xffffffffffffffff"),
				false,
			)
			require.NoError(t, err)
			require.Equal(t, result, IntFromString("340274119756928397675478831271437331477"))
		})

		t.Run("failing", func(t *testing.T) {
			_, err := amount1Delta(
				MinSqrtRatio,
				MaxSqrtRatio,
				U128Max,
				false,
			)
			require.Error(t, err)
		})
	})
}
