package validateroute

import "github.com/KyberNetwork/router-service/internal/pkg/core"

type IValidator interface {
	Validate(route core.Route) error
}

type ValidateRouteUseCase struct {
	validators []IValidator
}

func NewValidateRouteUseCase() *ValidateRouteUseCase {
	return &ValidateRouteUseCase{
		validators: make([]IValidator, 0),
	}
}

func (t *ValidateRouteUseCase) RegisterValidator(validator IValidator) {
	t.validators = append(t.validators, validator)
}

func (t *ValidateRouteUseCase) ValidateRouteResult(route core.Route) error {
	for _, v := range t.validators {
		err := v.Validate(route)
		if err != nil {
			return err
		}
	}

	return nil
}
