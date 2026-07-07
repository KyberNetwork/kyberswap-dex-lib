package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestGyroPoolMathSqrt(t *testing.T) {
	t.Parallel()
	t.Run("1. should return correct result", func(t *testing.T) {
		input := uint256.MustFromDecimal("4018432103219473921753291479214")
		expected := "2004602729525098177813955"
		actual, err := GyroPoolMath.Sqrt(input)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		input := uint256.MustFromDecimal("4890821048210147289147289142")
		expected := "69934405325348604762531"
		actual, err := GyroPoolMath.Sqrt(input)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		input := uint256.MustFromDecimal("48908210484124210147289147289142")
		expected := "6993440532679477224360521"
		actual, err := GyroPoolMath.Sqrt(input)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})
}
