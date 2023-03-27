package api

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/api/params"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/dto"
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
		span, ctx := tracer.StartSpanFromContext(ginCtx.Request.Context(), "GetPools")
		defer span.Finish()

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
		IDs: transformSliceParams(params.IDs),
	}
}
