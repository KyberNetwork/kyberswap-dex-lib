package business

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMaxAmountInExactInput(t *testing.T) {
	t.Run("it should return correct value", func(t *testing.T) {
		inputAmount := new(big.Int).SetInt64(1000000)
		slippageTolerance := new(big.Int).SetInt64(1500)

		maxAmountIn := GetMaxAmountInExactInput(inputAmount, slippageTolerance)

		assert.Equal(t, new(big.Int).SetInt64(1000000), maxAmountIn)
	})
}

func TestGetMaxAmountInExactOutput(t *testing.T) {
	t.Run("it should return correct value", func(t *testing.T) {
		testCases := []struct {
			inputAmount       *big.Int
			slippageTolerance *big.Int
			maxAmountIn       *big.Int
		}{
			{
				inputAmount:       new(big.Int).SetInt64(987654321),
				slippageTolerance: new(big.Int).SetInt64(1500),
				maxAmountIn:       new(big.Int).SetInt64(1135802469),
			},
			{
				inputAmount:       new(big.Int).SetInt64(123456),
				slippageTolerance: new(big.Int).SetInt64(2000),
				maxAmountIn:       new(big.Int).SetInt64(148147),
			},
		}

		for _, tc := range testCases {
			maxAmountIn := GetMaxAmountInExactOutput(tc.inputAmount, tc.slippageTolerance)

			assert.Equal(t, tc.maxAmountIn, maxAmountIn)
		}
	})
}
