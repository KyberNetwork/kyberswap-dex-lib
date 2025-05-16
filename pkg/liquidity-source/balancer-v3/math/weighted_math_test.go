package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestWeightedMath_ComputeOutGivenExactIn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		balanceIn      *uint256.Int
		weightIn       *uint256.Int
		balanceOut     *uint256.Int
		weightOut      *uint256.Int
		amountIn       *uint256.Int
		expectedAmount *uint256.Int
		expectedErr    error
	}{
		{
			name:           "Normal case",
			balanceIn:      uint256.MustFromDecimal("48251596353174400713087"),
			weightIn:       uint256.NewInt(500000000000000000),
			balanceOut:     uint256.MustFromDecimal("10798853824000000000000"),
			weightOut:      uint256.NewInt(500000000000000000),
			amountIn:       uint256.NewInt(1e18),
			expectedAmount: uint256.NewInt(223798399260422100),
			expectedErr:    nil,
		},
		{
			name:           "AmountIn exceeds MAX_IN_RATIO",
			balanceIn:      uint256.NewInt(1000),
			weightIn:       uint256.NewInt(50),
			balanceOut:     uint256.NewInt(500),
			weightOut:      uint256.NewInt(50),
			amountIn:       uint256.NewInt(301), // Exceeds MAX_IN_RATIO = 30% of balanceIn
			expectedAmount: nil,
			expectedErr:    ErrMaxInRatio,
		},

		{
			name:           "Zero amountIn",
			balanceIn:      uint256.NewInt(1000),
			weightIn:       uint256.NewInt(50),
			balanceOut:     uint256.NewInt(500),
			weightOut:      uint256.NewInt(50),
			amountIn:       uint256.NewInt(0),
			expectedAmount: uint256.NewInt(0),
			expectedErr:    nil,
		},
		{
			name:           "Invalid weights (weightOut == 0)",
			balanceIn:      uint256.NewInt(1000),
			weightIn:       uint256.NewInt(50),
			balanceOut:     uint256.NewInt(500),
			weightOut:      uint256.NewInt(0),
			amountIn:       uint256.NewInt(10),
			expectedAmount: uint256.NewInt(0),
			expectedErr:    ErrZeroDivision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := WeightedMath.ComputeOutGivenExactIn(
				tt.balanceIn,
				tt.weightIn,
				tt.balanceOut,
				tt.weightOut,
				tt.amountIn,
			)

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAmount, amount)
			}
		})
	}
}

func TestWeightedMath_ComputeInGivenExactOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		balanceIn      *uint256.Int
		weightIn       *uint256.Int
		balanceOut     *uint256.Int
		weightOut      *uint256.Int
		amountOut      *uint256.Int
		expectedAmount *uint256.Int
		expectedErr    error
	}{
		{
			name:           "Normal case",
			balanceIn:      uint256.MustFromDecimal("48251596353174400713087"),
			weightIn:       uint256.NewInt(500000000000000000),
			balanceOut:     uint256.MustFromDecimal("10798853824000000000000"),
			weightOut:      uint256.NewInt(500000000000000000),
			amountOut:      uint256.NewInt(1e17),
			expectedAmount: uint256.NewInt(446825598023519286),
			expectedErr:    nil,
		},
		{
			name:           "AmountOut exceeds MAX_OUT_RATIO",
			balanceIn:      uint256.NewInt(1000),
			weightIn:       uint256.NewInt(50),
			balanceOut:     uint256.NewInt(1000),
			weightOut:      uint256.NewInt(50),
			amountOut:      uint256.NewInt(301), // Exceeds MAX_OUT_RATIO = 30% of balanceOut
			expectedAmount: nil,
			expectedErr:    ErrMaxOutRatio,
		},
		{
			name:           "Zero amountOut",
			balanceIn:      uint256.NewInt(1000),
			weightIn:       uint256.NewInt(50),
			balanceOut:     uint256.NewInt(500),
			weightOut:      uint256.NewInt(50),
			amountOut:      uint256.NewInt(0),
			expectedAmount: uint256.NewInt(0),
			expectedErr:    nil,
		},
		{
			name:           "Invalid weights (weightIn == 0)",
			balanceIn:      uint256.NewInt(1000),
			weightIn:       uint256.NewInt(0),
			balanceOut:     uint256.NewInt(500),
			weightOut:      uint256.NewInt(50),
			amountOut:      uint256.NewInt(10),
			expectedAmount: uint256.NewInt(0),
			expectedErr:    ErrZeroDivision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := WeightedMath.ComputeInGivenExactOut(
				tt.balanceIn,
				tt.weightIn,
				tt.balanceOut,
				tt.weightOut,
				tt.amountOut,
			)

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAmount, amount)
			}
		})
	}
}
