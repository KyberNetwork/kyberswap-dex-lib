package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestAmount0(t *testing.T) {
	t.Parallel()
	t.Run("priceDown", func(t *testing.T) {
		result, err := Amount0Delta(
			bignum.NewBig("339942424496442021441932674757011200255"),
			TwoPow128,
			big.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big.NewInt(1_000))
	})

	t.Run("priceDownReverse", func(t *testing.T) {
		result, err := Amount0Delta(
			TwoPow128,
			bignum.NewBig("339942424496442021441932674757011200255"),
			big.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big.NewInt(1_000))
	})

	t.Run("priceExampleDown", func(t *testing.T) {
		result, err := Amount0Delta(
			TwoPow128,
			new(big.Int).Add(bignum.NewBig("34028236692093846346337460743176821145"), TwoPow128),
			bignum.NewBig("1_000_000_000_000_000_000"),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, bignum.NewBig("90_909_090_909_090_909"))
	})

	t.Run("priceExampleUp", func(t *testing.T) {
		result, err := Amount0Delta(
			TwoPow128,
			new(big.Int).Add(bignum.NewBig("34028236692093846346337460743176821145"), TwoPow128),
			bignum.NewBig("1_000_000_000_000_000_000"),
			true,
		)
		require.NoError(t, err)
		require.Equal(t, result, bignum.NewBig("90_909_090_909_090_910"))
	})
}

func TestAmount1(t *testing.T) {
	t.Parallel()
	t.Run("priceDown", func(t *testing.T) {
		result, err := Amount1Delta(
			bignum.NewBig("339942424496442021441932674757011200255"),
			TwoPow128,
			big.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big.NewInt(999))
	})

	t.Run("priceDownReverse", func(t *testing.T) {
		result, err := Amount1Delta(
			TwoPow128,
			bignum.NewBig("339942424496442021441932674757011200255"),
			big.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big.NewInt(999))
	})

	t.Run("priceUp", func(t *testing.T) {
		result, err := Amount1Delta(
			new(big.Int).Add(bignum.NewBig("340622989910849312776150758189957120"), TwoPow128),
			TwoPow128,
			big.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big.NewInt(1001))
	})

	t.Run("priceUpReverse", func(t *testing.T) {
		result, err := Amount1Delta(
			TwoPow128,
			bignum.NewBig("339942424496442021441932674757011200255"),
			big.NewInt(1_000_000),
			true,
		)
		require.NoError(t, err)
		require.Equal(t, result, big.NewInt(1000))
	})

	t.Run("priceExampleDown", func(t *testing.T) {
		result, err := Amount1Delta(
			TwoPow128,
			bignum.NewBig("309347606291762239512158734028880192232"),
			bignum.NewBig("1_000_000_000_000_000_000"),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, bignum.NewBig("90_909_090_909_090_909"))
	})

	t.Run("priceExampleUp", func(t *testing.T) {
		result, err := Amount1Delta(
			TwoPow128,
			bignum.NewBig("309347606291762239512158734028880192232"),
			bignum.NewBig("1_000_000_000_000_000_000"),
			true,
		)
		require.NoError(t, err)
		require.Equal(t, result, bignum.NewBig("90_909_090_909_090_910"))
	})

	t.Run("noOverflowHalfPriceRange", func(t *testing.T) {
		result, err := Amount1Delta(
			TwoPow128,
			MaxSqrtRatio,
			bignum.NewBig("0xffffffffffffffff"),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, bignum.NewBig("340274119756928397675478831269759003622"))
	})

	t.Run("failing", func(t *testing.T) {
		_, err := Amount1Delta(
			MinSqrtRatio,
			MaxSqrtRatio,
			U128Max,
			false,
		)
		require.Error(t, err)
	})
}
