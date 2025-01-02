package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestStableMath_ComputeInvariant(t *testing.T) {
	tests := []struct {
		name              string
		amp               *uint256.Int
		balances          []*uint256.Int
		expectedInvariant *uint256.Int
		expectedErr       error
	}{
		{
			name: "Basic 2 token pool",
			amp:  uint256.NewInt(1000000),
			balances: []*uint256.Int{
				uint256.NewInt(1000000),
				uint256.NewInt(1000000),
			},
			expectedInvariant: uint256.NewInt(2000000),
			expectedErr:       nil,
		},
		{
			name: "Imbalanced 2 token pool",
			amp:  uint256.NewInt(200000),
			balances: []*uint256.Int{
				uint256.MustFromDecimal("340867122491122140643"),
				uint256.MustFromDecimal("384610409069784884043"),
			},
			expectedInvariant: uint256.MustFromDecimal("725470946757739599230"),
			expectedErr:       nil,
		},
		{
			name: "3 token pool",
			amp:  uint256.NewInt(1000000),
			balances: []*uint256.Int{
				uint256.NewInt(1000000),
				uint256.NewInt(1000000),
				uint256.NewInt(1000000),
			},
			expectedInvariant: uint256.NewInt(3000000),
			expectedErr:       nil,
		},
		{
			name: "Zero balances",
			amp:  uint256.NewInt(1000000),
			balances: []*uint256.Int{
				uint256.NewInt(0),
				uint256.NewInt(0),
			},
			expectedInvariant: uint256.NewInt(0),
			expectedErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invariant, err := StableMath.ComputeInvariant(tt.amp, tt.balances)

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedInvariant, invariant)
			}
		})
	}
}

func TestStableMath_ComputeBalance(t *testing.T) {
	tests := []struct {
		name            string
		amp             *uint256.Int
		balances        []*uint256.Int
		invariant       *uint256.Int
		tokenIndex      int
		expectedBalance *uint256.Int
		expectedErr     error
	}{
		{
			name: "Balanced 2 token pool",
			amp:  uint256.NewInt(1000000),
			balances: []*uint256.Int{
				uint256.NewInt(1000000),
				uint256.NewInt(1000000),
			},
			invariant:       uint256.NewInt(2000000),
			tokenIndex:      0,
			expectedBalance: uint256.NewInt(1000001),
			expectedErr:     nil,
		},
		{
			name: "Imbalanced 2 token pool",
			amp:  uint256.NewInt(200000),
			balances: []*uint256.Int{
				uint256.MustFromDecimal("340867122491122140643"),
				uint256.MustFromDecimal("384610409069784884043"),
			},
			invariant:       uint256.MustFromDecimal("725470946757739599230"),
			tokenIndex:      1,
			expectedBalance: uint256.MustFromDecimal("384610409069784884044"),
			expectedErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := StableMath.ComputeBalance(tt.amp, tt.balances, tt.invariant, tt.tokenIndex)

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBalance, balance)
			}
		})
	}
}

func TestStableMath_ComputeOutGivenExactIn(t *testing.T) {
	tests := []struct {
		name           string
		amp            *uint256.Int
		balances       []*uint256.Int
		tokenIndexIn   int
		tokenIndexOut  int
		tokenAmountIn  *uint256.Int
		invariant      *uint256.Int
		expectedAmount *uint256.Int
		expectedErr    error
	}{
		{
			name: "Equal pool swap",
			amp:  uint256.NewInt(1000000),
			balances: []*uint256.Int{
				uint256.NewInt(1000000),
				uint256.NewInt(1000000),
			},
			tokenIndexIn:   0,
			tokenIndexOut:  1,
			tokenAmountIn:  uint256.NewInt(100),
			invariant:      uint256.NewInt(2000000),
			expectedAmount: uint256.NewInt(98),
			expectedErr:    nil,
		},
		{
			name: "Imbalanced pool swap",
			amp:  uint256.NewInt(100),
			balances: []*uint256.Int{
				uint256.NewInt(1500000),
				uint256.NewInt(500000),
			},
			tokenIndexIn:   0,
			tokenIndexOut:  1,
			tokenAmountIn:  uint256.NewInt(100),
			invariant:      uint256.NewInt(1000000),
			expectedAmount: uint256.NewInt(352454),
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := StableMath.ComputeOutGivenExactIn(
				tt.amp,
				tt.balances,
				tt.tokenIndexIn,
				tt.tokenIndexOut,
				tt.tokenAmountIn,
				tt.invariant,
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

func TestStableMath_ComputeInGivenExactOut(t *testing.T) {
	tests := []struct {
		name           string
		amp            *uint256.Int
		balances       []*uint256.Int
		tokenIndexIn   int
		tokenIndexOut  int
		tokenAmountIn  *uint256.Int
		invariant      *uint256.Int
		expectedAmount *uint256.Int
		expectedErr    error
	}{
		{
			name: "Equal pool swap",
			amp:  uint256.NewInt(1000000),
			balances: []*uint256.Int{
				uint256.NewInt(1000000),
				uint256.NewInt(1000000),
			},
			tokenIndexIn:   0,
			tokenIndexOut:  1,
			tokenAmountIn:  uint256.NewInt(100),
			invariant:      uint256.NewInt(2000000),
			expectedAmount: uint256.NewInt(98),
			expectedErr:    nil,
		},
		{
			name: "Imbalanced pool swap",
			amp:  uint256.NewInt(100),
			balances: []*uint256.Int{
				uint256.NewInt(1500000),
				uint256.NewInt(500000),
			},
			tokenIndexIn:   0,
			tokenIndexOut:  1,
			tokenAmountIn:  uint256.NewInt(100),
			invariant:      uint256.NewInt(1000000),
			expectedAmount: uint256.NewInt(352454),
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := StableMath.ComputeOutGivenExactIn(
				tt.amp,
				tt.balances,
				tt.tokenIndexIn,
				tt.tokenIndexOut,
				tt.tokenAmountIn,
				tt.invariant,
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
