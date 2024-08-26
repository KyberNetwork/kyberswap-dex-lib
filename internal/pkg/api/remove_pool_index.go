package api

import (
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

var MAX_ADDRESSES = 100

func RemovePoolsFromIndex(usecase IRemovePoolIndexUseCase) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {

		var reqParams params.RemovePoolIndexParams
		if err := ctx.ShouldBindJSON(&reqParams); err != nil {
			RespondFailure(ctx, err)
			return
		}

		if len(reqParams.Addresses) == 0 {
			RespondFailure(ctx, validator.NewValidationError("pools", "required"))
			return
		}

		addresses := utils.TransformSliceParams(reqParams.Addresses)
		if len(addresses) > MAX_ADDRESSES {
			RespondFailure(ctx, validator.NewValidationError("pools", fmt.Sprintf("exceed maximum value %d", MAX_ADDRESSES)))
			return
		}
		err := usecase.RemovePoolAddressFromIndexes(ctx, addresses)
		if err != nil {
			RespondFailure(ctx, err)
			return
		}

		RespondSuccess(ctx, nil)
	}
}
