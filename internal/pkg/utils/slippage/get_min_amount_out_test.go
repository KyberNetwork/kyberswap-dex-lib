package slippage

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
)

func TestGetMinAmountOutExactInput(t *testing.T) {
	t.Run("it should return correct value", func(t *testing.T) {
		testCases := []struct {
			outputAmount      *big.Int
			slippageTolerance float64
			minAmountOut      *big.Int
		}{
			{
				outputAmount:      bignumber.NewBig10("68409610350685460046340"),
				slippageTolerance: 0.01,
				minAmountOut:      bignumber.NewBig10("68409541941075109360879"),
			},
			{
				outputAmount:      new(big.Int).SetInt64(987654321),
				slippageTolerance: 1500,
				minAmountOut:      new(big.Int).SetInt64(839506172),
			},
			{
				outputAmount:      new(big.Int).SetInt64(123456),
				slippageTolerance: 2000,
				minAmountOut:      new(big.Int).SetInt64(98764),
			},
			{
				outputAmount:      new(big.Int).SetInt64(1),
				slippageTolerance: 50,
				minAmountOut:      new(big.Int).SetInt64(1),
			},
			{
				outputAmount:      new(big.Int).SetInt64(1),
				slippageTolerance: 100,
				minAmountOut:      new(big.Int).SetInt64(1),
			},
			{
				outputAmount:      new(big.Int).SetInt64(2),
				slippageTolerance: 1000,
				minAmountOut:      new(big.Int).SetInt64(1),
			},
			{
				outputAmount:      new(big.Int).SetInt64(1),
				slippageTolerance: 0,
				minAmountOut:      new(big.Int).SetInt64(1),
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
		slippageTolerance := float64(1500)

		minAmountOut := GetMinAmountOutExactOutput(outputAmount, slippageTolerance)

		assert.Equal(t, new(big.Int).SetInt64(1000000), minAmountOut)
	})
}
