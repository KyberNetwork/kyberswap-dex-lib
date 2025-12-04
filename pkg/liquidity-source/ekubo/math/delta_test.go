package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestAmount0(t *testing.T) {
	t.Parallel()
	t.Run("priceDown", func(t *testing.T) {
		result, err := Amount0Delta(
			big256.New("339942424496442021441932674757011200255"),
			big256.U2Pow128,
			uint256.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, uint256.NewInt(1_000))
	})

	t.Run("priceDownReverse", func(t *testing.T) {
		result, err := Amount0Delta(
			big256.U2Pow128,
			big256.New("339942424496442021441932674757011200255"),
			uint256.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, uint256.NewInt(1_000))
	})

	t.Run("priceExampleDown", func(t *testing.T) {
		result, err := Amount0Delta(
			big256.U2Pow128,
			new(uint256.Int).Add(big256.New("34028236692093846346337460743176821145"), big256.U2Pow128),
			big256.New("1000000000000000000"),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big256.New("90909090909090909"))
	})

	t.Run("priceExampleUp", func(t *testing.T) {
		result, err := Amount0Delta(
			big256.U2Pow128,
			new(uint256.Int).Add(big256.New("34028236692093846346337460743176821145"), big256.U2Pow128),
			big256.New("1000000000000000000"),
			true,
		)
		require.NoError(t, err)
		require.Equal(t, result, big256.New("90909090909090910"))
	})
}

func TestAmount1(t *testing.T) {
	t.Parallel()
	t.Run("priceDown", func(t *testing.T) {
		result, err := Amount1Delta(
			big256.New("339942424496442021441932674757011200255"),
			big256.U2Pow128,
			uint256.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, uint256.NewInt(999))
	})

	t.Run("priceDownReverse", func(t *testing.T) {
		result, err := Amount1Delta(
			big256.U2Pow128,
			big256.New("339942424496442021441932674757011200255"),
			uint256.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, uint256.NewInt(999))
	})

	t.Run("priceUp", func(t *testing.T) {
		result, err := Amount1Delta(
			new(uint256.Int).Add(big256.New("340622989910849312776150758189957120"), big256.U2Pow128),
			big256.U2Pow128,
			uint256.NewInt(1_000_000),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, uint256.NewInt(1001))
	})

	t.Run("priceUpReverse", func(t *testing.T) {
		result, err := Amount1Delta(
			big256.U2Pow128,
			big256.New("339942424496442021441932674757011200255"),
			uint256.NewInt(1_000_000),
			true,
		)
		require.NoError(t, err)
		require.Equal(t, result, uint256.NewInt(1000))
	})

	t.Run("priceExampleDown", func(t *testing.T) {
		result, err := Amount1Delta(
			big256.U2Pow128,
			big256.New("309347606291762239512158734028880192232"),
			big256.New("1000000000000000000"),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big256.New("90909090909090909"))
	})

	t.Run("priceExampleUp", func(t *testing.T) {
		result, err := Amount1Delta(
			big256.U2Pow128,
			big256.New("309347606291762239512158734028880192232"),
			big256.New("1000000000000000000"),
			true,
		)
		require.NoError(t, err)
		require.Equal(t, result, big256.New("90909090909090910"))
	})

	t.Run("noOverflowHalfPriceRange", func(t *testing.T) {
		result, err := Amount1Delta(
			big256.U2Pow128,
			MaxSqrtRatio,
			uint256.MustFromHex("0xffffffffffffffff"),
			false,
		)
		require.NoError(t, err)
		require.Equal(t, result, big256.New("340274119756928397675478831269759003622"))
	})

	t.Run("failing", func(t *testing.T) {
		_, err := Amount1Delta(
			MinSqrtRatio,
			MaxSqrtRatio,
			big256.UMaxU128,
			false,
		)
		require.Error(t, err)
	})
}
