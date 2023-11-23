package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func Test_calcOutGivenIn(t *testing.T) {
	t.Run("1.should return OK", func(t *testing.T) {
		// input
		balanceIn := uint256.MustFromDecimal("2133741937219414819371293")
		weightIn := uint256.MustFromDecimal("10")
		balanceOut := uint256.MustFromDecimal("548471973423647283412313")
		weightOut := uint256.MustFromDecimal("20")
		amountIn := uint256.MustFromDecimal("21481937129313123729")

		// expected
		expected := "2760912942840907991"

		// calculation
		result, err := WeightedMath.CalcOutGivenIn(balanceIn, weightIn, balanceOut, weightOut, amountIn)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("2.should return OK", func(t *testing.T) {
		// input
		balanceIn := uint256.MustFromDecimal("92174932794319461529478329")
		weightIn := uint256.MustFromDecimal("15")
		balanceOut := uint256.MustFromDecimal("2914754379179427149231562")
		weightOut := uint256.MustFromDecimal("5")
		amountIn := uint256.MustFromDecimal("14957430248210")

		// expected
		expected := "1389798609308"

		// calculation
		result, err := WeightedMath.CalcOutGivenIn(balanceIn, weightIn, balanceOut, weightOut, amountIn)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("3.should return error exceed amount in ratio", func(t *testing.T) {
		// input
		balanceIn := uint256.MustFromDecimal("92174932794319461529478329")
		weightIn := uint256.MustFromDecimal("15")
		balanceOut := uint256.MustFromDecimal("2914754379179427149231562")
		weightOut := uint256.MustFromDecimal("5")
		amountIn := uint256.MustFromDecimal("92174932794319461529478329")

		// calculation
		_, err := WeightedMath.CalcOutGivenIn(balanceIn, weightIn, balanceOut, weightOut, amountIn)

		// assert
		assert.ErrorIs(t, err, ErrMaxInRatio)
	})
}
