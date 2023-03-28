package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
)

func TestGetPoolsParamsValidator_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		params params.GetPoolsParams
		err    error
	}{
		{
			name:   "it should return correct error when ids is empty",
			params: params.GetPoolsParams{},
			err:    NewValidationError("ids", "required"),
		},
		{
			name:   "it should return nil when params is valid",
			params: params.GetPoolsParams{IDs: "0x9f5c637a4112c6c5450ca0fa02fcb357f4e100d5"},
			err:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewGetPoolsParamsValidator()

			err := validator.Validate(tc.params)

			assert.Equal(t, tc.err, err)
		})
	}
}
