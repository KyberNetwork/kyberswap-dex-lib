package core

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcAmountUSD(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		amount            *big.Int
		decimals          uint8
		price             float64
		expectedAmountUSD float64
	}{
		{
			name:              "it should return correct amountUSD",
			amount:            big.NewInt(1000000),
			decimals:          6,
			price:             1.2,
			expectedAmountUSD: 1.2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountUSD := CalcAmountUSD(tc.amount, tc.decimals, tc.price)

			amountUSDFl, _ := amountUSD.Float64()

			assert.InDelta(t, tc.expectedAmountUSD, amountUSDFl, 0.01)
		})
	}
}
