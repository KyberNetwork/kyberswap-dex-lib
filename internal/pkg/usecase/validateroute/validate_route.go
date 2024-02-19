package validateroute

import (
	"context"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IValidator interface {
	Validate(ctx context.Context, poolByAddress map[string]poolpkg.IPoolSimulator, route *valueobject.Route) error
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

func (t *ValidateRouteUseCase) ValidateRouteResult(ctx context.Context, poolByAddress map[string]poolpkg.IPoolSimulator, route *valueobject.Route) error {
	for _, v := range t.validators {
		err := v.Validate(ctx, poolByAddress, route)
		if err != nil {
			return err
		}
	}

	return nil
}
