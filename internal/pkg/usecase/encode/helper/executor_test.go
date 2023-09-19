package helper

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateMinimumPSAmountOut(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		amountOut         *big.Int
		expectedMinimumPS *big.Int
	}{
		{
			name:              "it should calculate minimum PS amount out correctly",
			amountOut:         big.NewInt(2729797571728140385),
			expectedMinimumPS: big.NewInt(2729797571728),
		},
		{
			name:              "it should fallback to 1 if calculated minimum PS is too small",
			amountOut:         big.NewInt(100),
			expectedMinimumPS: big.NewInt(1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetMinPositiveSlippageAmount(tc.amountOut, 1000000)
			assert.Equal(t, tc.expectedMinimumPS, result)
		})
	}
}
