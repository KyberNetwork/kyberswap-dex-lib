package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlippageValidator_validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                 string
		slippageTolerance    int64
		ignoreCappedSlippage bool
		err                  error
	}{
		{
			name:              "it should return [slippageTolerance|invalid]",
			slippageTolerance: 5001,
			err:               NewValidationError("slippageTolerance", "invalid"),
		},
		{
			name:              "it should return [slippageTolerance|invalid]",
			slippageTolerance: -1,
			err:               NewValidationError("slippageTolerance", "invalid"),
		},
		{
			name:                 "it should return nil due to ignore slipapge validate checking",
			slippageTolerance:    -1,
			ignoreCappedSlippage: true,
			err:                  nil,
		},
		{
			name:                 "it should return nil due to ignore slipapge validate checking although slippage is above slippage gte",
			slippageTolerance:    5001,
			ignoreCappedSlippage: true,
			err:                  nil,
		},
		{
			name:              "it should return nil",
			slippageTolerance: 1000,
			err:               nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewSlippageValidator(
				SlippageValidatorConfig{
					SlippageToleranceGTE: 0,
					SlippageToleranceLTE: 5000,
				},
			)

			err := validator.Validate(tc.slippageTolerance, tc.ignoreCappedSlippage)

			assert.Equal(t, tc.err, err)
		})
	}
}
