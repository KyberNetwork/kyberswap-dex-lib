package validator

import (
	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
)

type getPoolsParamsValidator struct{}

func NewGetPoolsParamsValidator() *getPoolsParamsValidator {
	return &getPoolsParamsValidator{}
}

func (v *getPoolsParamsValidator) Validate(params params.GetPoolsParams) error {
	return v.validateIDs(params.IDs)
}

func (v *getPoolsParamsValidator) validateIDs(ids string) error {
	if len(ids) == 0 {
		return NewValidationError("ids", "required")
	}

	return nil
}
