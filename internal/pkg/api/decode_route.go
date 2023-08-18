package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
)

// DecodeSwapData [POST /debug/decode] decode built route
func DecodeSwapData(useCase IDecodeSwapDataUseCase) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		var bodyParams params.DecodeSwapDataParams
		if err := ginCtx.ShouldBindJSON(&bodyParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.Wrapf(
					ErrBindRequestBodyFailed,
					"[DecodeRoute] err: [%v]", err),
			)
			return
		}

		result, err := useCase.Decode(bodyParams.EncodedData)
		if err != nil {
			RespondFailure(
				ginCtx,
				errors.Wrapf(
					ErrInvalidValue,
					"[DecodeRoute] err: [%v]", err),
			)
			return
		}

		RespondSuccess(ginCtx, result)
	}
}
