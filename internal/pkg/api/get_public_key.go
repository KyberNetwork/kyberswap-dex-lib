package api

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func GetPublicKey(useCase IGetPublicKeyUseCase) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromContext(ginCtx.Request.Context(), "GetPublicKey")
		defer span.Finish()

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
