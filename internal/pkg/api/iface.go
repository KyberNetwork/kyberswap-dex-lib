package api

//go:generate mockgen -destination ../mocks/api/get_pools_use_case.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IGetPoolsUseCase
//go:generate mockgen -destination ../mocks/api/get_tokens_use_case.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IGetTokensUseCase
//go:generate mockgen -destination ../mocks/api/get_routes_use_case.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IGetRoutesUseCase
//go:generate mockgen -destination ../mocks/api/build_route_use_case.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IBuildRouteUseCase
//go:generate mockgen -destination ../mocks/api/get_public_key.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IGetPublicKeyUseCase

//go:generate mockgen -destination ../mocks/api/get_pools_params_validator.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IGetPoolsParamsValidator
//go:generate mockgen -destination ../mocks/api/get_tokens_params_validator.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IGetTokensParamsValidator
//go:generate mockgen -destination ../mocks/api/get_routes_params_validator.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IGetRoutesParamsValidator
//go:generate mockgen -destination ../mocks/api/build_route_params_validator.go -package api github.com/KyberNetwork/router-service/internal/pkg/api IBuildRouteParamsValidator

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
)

// IGetPoolsUseCase is a use-case which handles getting pools logic
type IGetPoolsUseCase interface {
	Handle(ctx context.Context, query dto.GetPoolsQuery) (*dto.GetPoolsResult, error)
}

// IGetTokensUseCase is a use-case which handles getting tokens logic
type IGetTokensUseCase interface {
	Handle(ctx context.Context, query dto.GetTokensQuery) (*dto.GetTokensResult, error)
}

// IGetRoutesUseCase is a use-case which handles getting routes logic
type IGetRoutesUseCase interface {
	Handle(ctx context.Context, query dto.GetRoutesQuery) (*dto.GetRoutesResult, error)
}

// IBuildRouteUseCase is a use-case which handles building route logic
type IBuildRouteUseCase interface {
	Handle(ctx context.Context, command dto.BuildRouteCommand) (*dto.BuildRouteResult, error)
}

// IGetPublicKeyUseCase a use-case which handles getting public key.
type IGetPublicKeyUseCase interface {
	Handle(ctx context.Context, keyID string) (*dto.GetPublicKeyResult, error)
}

// IGetPoolsParamsValidator validates params.GetPoolsParams
type IGetPoolsParamsValidator interface {
	Validate(params params.GetPoolsParams) error
}

// IGetTokensParamsValidator validates params.GetTokensParams
type IGetTokensParamsValidator interface {
	Validate(params params.GetTokensParams) error
}

// IGetRoutesParamsValidator validates params.GetRoutesParams
type IGetRoutesParamsValidator interface {
	Validate(params params.GetRoutesParams) error
}

// IBuildRouteParamsValidator validates params.BuildRouteParams
type IBuildRouteParamsValidator interface {
	Validate(params params.BuildRouteParams) error
}
