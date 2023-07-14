package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
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
				errors.Wrapf(ErrBindQueryFailed, "[GetCustomRoutes] err: [%v]", err),
			)
			return
		}

		if err := validator.Validate(queryParams.GetRoutesParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		query, err := transformGetCustomRoutesParams(queryParams)
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

func transformGetCustomRoutesParams(params params.GetCustomRoutesParams) (dto.GetCustomRoutesQuery, error) {
	query, err := transformGetRoutesParams(params.GetRoutesParams)
	if err != nil {
		return dto.GetCustomRoutesQuery{}, err
	}

	return dto.GetCustomRoutesQuery{
		GetRoutesQuery: query,
		PoolIds:        transformSliceParams(params.PoolIds),
	}, nil
}
