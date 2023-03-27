package validator

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/api/params"
)

type getTokensParamsValidator struct{}

func NewGetTokensParamsValidator() *getTokensParamsValidator {
	return &getTokensParamsValidator{}
}

func (v *getTokensParamsValidator) Validate(params params.GetTokensParams) error {
	return v.validateIDs(params.IDs)
}

func (v *getTokensParamsValidator) validateIDs(ids string) error {
	if len(ids) == 0 {
		return NewValidationError("ids", "required")
	}

	return nil
}
