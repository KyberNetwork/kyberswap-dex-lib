package business

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcL1FeeUSD(t *testing.T) {
	testCases := []struct {
		name             string
		l1GasFee         *big.Int
		gasTokenPriceUSD float64
		expectedL1FeeUSD float64
	}{
		{
			name:             "it should return correct l1FeeUSD",
			l1GasFee:         big.NewInt(505952931733752),
			gasTokenPriceUSD: 1700,
			expectedL1FeeUSD: 0.86011,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gasUSD := CalcL1FeeUSD(tc.l1GasFee, tc.gasTokenPriceUSD)

			gasUSDFl, _ := gasUSD.Float64()

			assert.InDelta(t, tc.expectedL1FeeUSD, gasUSDFl, 0.01)
		})
	}
}
