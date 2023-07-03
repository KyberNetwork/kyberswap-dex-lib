package business

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMinAmountOutExactInput(t *testing.T) {
	t.Run("it should return correct value", func(t *testing.T) {
		testCases := []struct {
			outputAmount      *big.Int
			slippageTolerance *big.Int
			minAmountOut      *big.Int
		}{
			{
				outputAmount:      new(big.Int).SetInt64(987654321),
				slippageTolerance: new(big.Int).SetInt64(1500),
				minAmountOut:      new(big.Int).SetInt64(839506172),
			},
			{
				outputAmount:      new(big.Int).SetInt64(123456),
				slippageTolerance: new(big.Int).SetInt64(2000),
				minAmountOut:      new(big.Int).SetInt64(98764),
			},
		}

		for _, tc := range testCases {
			minAmountOut := GetMinAmountOutExactInput(tc.outputAmount, tc.slippageTolerance)

			assert.Equal(t, tc.minAmountOut, minAmountOut)
		}
	})
}

func TestGetMinAmountOutExactOutput(t *testing.T) {
	t.Run("it should return correct value", func(t *testing.T) {
		outputAmount := new(big.Int).SetInt64(1000000)
		slippageTolerance := new(big.Int).SetInt64(1500)

		minAmountOut := GetMinAmountOutExactOutput(outputAmount, slippageTolerance)

		assert.Equal(t, new(big.Int).SetInt64(1000000), minAmountOut)
	})
}
