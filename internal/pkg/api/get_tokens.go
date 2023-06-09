package api

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
)

// GetTokens [GET /tokens] Get tokens data
// ---
// parameters:
//   - ids: list of token addresses, separated by comma
func GetTokens(
	validator IGetTokensParamsValidator,
	useCase IGetTokensUseCase,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromContext(ginCtx.Request.Context(), "GetTokens")
		defer span.Finish()

		var queryParams params.GetTokensParams
		if err := ginCtx.ShouldBindQuery(&queryParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		if err := validator.Validate(queryParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		query := transformGetTokensParams(queryParams)

		result, err := useCase.Handle(ctx, query)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		RespondSuccess(ginCtx, result)
	}
}

func transformGetTokensParams(params params.GetTokensParams) dto.GetTokensQuery {
	return dto.GetTokensQuery{
		IDs: transformSliceParams(params.IDs),
	}
}
