package api

import (
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/gin-gonic/gin"
)

func GetPublicKey(useCase IGetPublicKeyUseCase) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromContext(ginCtx.Request.Context(), "GetPublicKey")
		defer span.End()

		result, err := useCase.Handle(ctx, getKeyIDFromParamRequest(ginCtx))
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}
		RespondSuccess(ginCtx, result)
	}
}

func getKeyIDFromParamRequest(ctx *gin.Context) string {
	return ctx.Param("keyId")
}
