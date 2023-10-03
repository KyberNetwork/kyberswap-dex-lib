package api

import (
	"math/big"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GetRoutes [GET /routes] Find best routes
func GetRoutes(
	validator IGetRoutesParamsValidator,
	useCase IGetRoutesUseCase,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromContext(ginCtx.Request.Context(), "GetRoutes")
		defer span.End()

		span.SetTag("request-uri", ginCtx.Request.URL.RequestURI())

		var queryParams params.GetRoutesParams
		if err := ginCtx.ShouldBindQuery(&queryParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.Wrapf(ErrBindQueryFailed, "[GetRoutes] err: [%v]", err),
			)
			return
		}

		if err := validator.Validate(queryParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		query, err := transformGetRoutesParams(queryParams)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		result, err := useCase.Handle(ctx, query)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		response := transformGetRoutesResult(result)

		RespondSuccess(ginCtx, response)
	}
}

func transformGetRoutesParams(params params.GetRoutesParams) (dto.GetRoutesQuery, error) {
	amountIn, ok := new(big.Int).SetString(params.AmountIn, 10)
	if !ok {
		return dto.GetRoutesQuery{}, errors.Wrapf(
			ErrInvalidValue,
			"amountIn: [%s]",
			params.AmountIn,
		)
	}

	var gasPrice *big.Float
	if params.GasPrice != "" {
		gasPrice, ok = new(big.Float).SetString(params.GasPrice)
		if !ok {
			return dto.GetRoutesQuery{}, errors.Wrapf(
				ErrInvalidValue,
				"gasPrice: [%s]",
				params.GasPrice,
			)
		}
	}

	extraFee := valueobject.ZeroExtraFee
	if params.FeeAmount != "" {
		feeAmount, ok := new(big.Int).SetString(params.FeeAmount, 10)
		if !ok {
			return dto.GetRoutesQuery{}, errors.Wrapf(
				ErrInvalidValue,
				"feeAmount: [%s]",
				params.FeeAmount,
			)
		}

		extraFee = valueobject.ExtraFee{
			FeeAmount:   feeAmount,
			ChargeFeeBy: valueobject.ChargeFeeBy(params.ChargeFeeBy),
			IsInBps:     params.IsInBps,
			FeeReceiver: params.FeeReceiver,
		}

		actualFeeAmount := extraFee.CalcActualFeeAmount(amountIn)

		if extraFee.IsChargeFeeByCurrencyIn() && actualFeeAmount.Cmp(amountIn) > 0 {
			return dto.GetRoutesQuery{}, errors.Wrapf(
				ErrFeeAmountGreaterThanAmountIn,
				"feeAmount: [%s], amountIn: [%s]",
				actualFeeAmount.String(),
				amountIn.String(),
			)
		}
	}

	return dto.GetRoutesQuery{
		TokenIn:                utils.CleanUpParam(params.TokenIn),
		TokenOut:               utils.CleanUpParam(params.TokenOut),
		AmountIn:               amountIn,
		IncludedSources:        utils.TransformSliceParams(params.IncludedSources),
		ExcludedSources:        utils.TransformSliceParams(params.ExcludedSources),
		SaveGas:                params.SaveGas,
		GasInclude:             params.GasInclude,
		GasPrice:               gasPrice,
		ExtraFee:               extraFee,
		IsPathGeneratorEnabled: params.IsPathGeneratorEnabled,
	}, nil
}

func transformGetRoutesResult(result *dto.GetRoutesResult) *params.GetRoutesResponse {
	if result == nil {
		return nil
	}

	return &params.GetRoutesResponse{
		RouteSummary:  transformRouteSummary(result.RouteSummary),
		RouterAddress: result.RouterAddress,
	}
}

func transformRouteSummary(routeSummary *valueobject.RouteSummary) *params.RouteSummary {
	if routeSummary == nil {
		return nil
	}

	return &params.RouteSummary{
		TokenIn:                     routeSummary.TokenIn,
		AmountIn:                    routeSummary.AmountIn.String(),
		AmountInUSD:                 strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),
		TokenInMarketPriceAvailable: routeSummary.TokenInMarketPriceAvailable,

		TokenOut:                     routeSummary.TokenOut,
		AmountOut:                    routeSummary.AmountOut.String(),
		AmountOutUSD:                 strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),
		TokenOutMarketPriceAvailable: routeSummary.TokenOutMarketPriceAvailable,

		Gas:      strconv.FormatInt(routeSummary.Gas, 10),
		GasPrice: routeSummary.GasPrice.Text('f', -1),
		GasUSD:   strconv.FormatFloat(routeSummary.GasUSD, 'f', -1, 64),

		ExtraFee: transformExtraFee(routeSummary.ExtraFee),
		Route:    transformRoute(routeSummary.Route),
	}
}

func transformExtraFee(extraFee valueobject.ExtraFee) params.ExtraFee {
	return params.ExtraFee{
		FeeAmount:   extraFee.FeeAmount.String(),
		ChargeFeeBy: string(extraFee.ChargeFeeBy),
		FeeReceiver: extraFee.FeeReceiver,
		IsInBps:     extraFee.IsInBps,
	}
}

func transformRoute(route [][]valueobject.Swap) [][]params.Swap {
	routeParams := make([][]params.Swap, 0, len(route))

	for _, path := range route {
		pathParams := make([]params.Swap, 0, len(path))

		for _, swap := range path {
			pathParams = append(pathParams, transformSwap(swap))
		}

		routeParams = append(routeParams, pathParams)
	}

	return routeParams
}

func transformSwap(swap valueobject.Swap) params.Swap {
	return params.Swap{
		Pool:              swap.Pool,
		TokenIn:           swap.TokenIn,
		TokenOut:          swap.TokenOut,
		LimitReturnAmount: swap.LimitReturnAmount.String(),
		SwapAmount:        swap.SwapAmount.String(),
		AmountOut:         swap.AmountOut.String(),
		Exchange:          string(swap.Exchange),
		PoolLength:        swap.PoolLength,
		PoolType:          swap.PoolType,
		PoolExtra:         swap.PoolExtra,
		Extra:             swap.Extra,
	}
}
