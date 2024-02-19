package api

import (
	"github.com/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
)

// GetPools [GET /pools] Get pools data
// ---
// parameters:
//   - ids: list of pool addresses, separated by comma
func GetPools(
	validator IGetPoolsParamsValidator,
	useCase IGetPoolsUseCase,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromGinContext(ginCtx, "GetPools")
		defer span.End()

		var reqParams params.GetPoolsParams
		if err := ginCtx.ShouldBindQuery(&reqParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		if err := validator.Validate(reqParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		query := transformGetPoolsParams(reqParams)

		result, err := useCase.Handle(ctx, query)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		RespondSuccess(ginCtx, result)
	}
}

func transformGetPoolsParams(params params.GetPoolsParams) dto.GetPoolsQuery {
	return dto.GetPoolsQuery{
		IDs: utils.TransformSliceParams(params.IDs),
	}
}
