package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

// GetCustomRoutes [GET /custom-routes] Find best routes within input pools
func GetCustomRoutes(
	validator IGetRoutesParamsValidator,
	useCase IGetCustomRoutesUseCase,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		var queryParams params.GetCustomRoutesParams
		if err := ginCtx.ShouldBindQuery(&queryParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.WithMessagef(ErrBindQueryFailed, "[GetCustomRoutes] err: [%v]", err),
			)
			return
		}

		if err := validator.Validate(queryParams.GetRoutesParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		query, err := transformGetCustomRoutesParams(ginCtx, queryParams)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		result, err := useCase.Handle(ginCtx, query)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		response := transformGetRoutesResult(result)

		RespondSuccess(ginCtx, response)
	}
}

func transformGetCustomRoutesParams(ginCtx *gin.Context, params params.GetCustomRoutesParams) (dto.GetCustomRoutesQuery,
	error) {
	query, err := transformGetRoutesParams(ginCtx, params.GetRoutesParams)
	if err != nil {
		return dto.GetCustomRoutesQuery{}, err
	}

	return dto.GetCustomRoutesQuery{
		GetRoutesQuery: query,
		PoolIds:        utils.TransformSliceParams(params.PoolIds),
		EnableAlphaFee: params.EnableAlphaFee,
	}, nil
}
