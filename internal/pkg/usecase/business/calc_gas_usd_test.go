package business

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcGasUSD(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		gasPrice         *big.Float
		totalGas         int64
		gasTokenPriceUSD float64
		expectedGasUSD   float64
	}{
		{
			name:             "it should return correct amountUSD",
			gasPrice:         big.NewFloat(10000000000),
			totalGas:         400000,
			gasTokenPriceUSD: 1200,
			expectedGasUSD:   4.8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gasUSD := CalcGasUSD(tc.gasPrice, tc.totalGas, tc.gasTokenPriceUSD)

			gasUSDFl, _ := gasUSD.Float64()

			assert.InDelta(t, tc.expectedGasUSD, gasUSDFl, 0.01)
		})
	}
}
