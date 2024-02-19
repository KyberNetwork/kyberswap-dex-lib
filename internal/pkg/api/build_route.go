package api

import (
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// BuildRoute [POST /route/build] build route
func BuildRoute(
	validator IBuildRouteParamsValidator,
	useCase IBuildRouteUseCase,
	nowFunc func() time.Time,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromGinContext(ginCtx, "BuildRoute")
		defer span.End()

		clientIDFromHeader := clientid.ExtractClientID(ginCtx)

		var bodyParams params.BuildRouteParams
		if err := ginCtx.ShouldBindJSON(&bodyParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.Wrapf(
					ErrBindRequestBodyFailed,
					"[BuildRoute] err: [%v]", err),
			)
			return
		}

		// if source param is empty, use clientID from header as the source
		if bodyParams.Source == "" {
			bodyParams.Source = clientIDFromHeader
		}

		if err := validator.Validate(ctx, bodyParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		command, err := transformBuildRouteParams(bodyParams, nowFunc)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		result, err := useCase.Handle(ctx, command)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		RespondSuccess(ginCtx, result)
	}
}

// transformBuildRouteParams transform params.BuildRouteParams to dto.BuildRouteCommand
func transformBuildRouteParams(params params.BuildRouteParams, nowFunc func() time.Time) (dto.BuildRouteCommand, error) {
	routeSummary, err := transformRouteSummaryParams(params.RouteSummary)
	if err != nil {
		return dto.BuildRouteCommand{}, err
	}

	deadline := params.Deadline
	if params.Deadline == 0 {
		deadline = nowFunc().Add(valueobject.DefaultDeadline).Unix()
	}

	permit := common.FromHex(params.Permit)

	return dto.BuildRouteCommand{
		RouteSummary:        routeSummary,
		Deadline:            deadline,
		SlippageTolerance:   params.SlippageTolerance,
		Recipient:           params.Recipient,
		Referral:            params.Referral,
		Source:              params.Source,
		Sender:              params.Sender,
		Permit:              permit,
		EnableGasEstimation: params.EnableGasEstimation,
	}, nil
}

// transformRouteSummaryParams transforms params.RouteSummary to valueobject.RouteSummary
func transformRouteSummaryParams(params params.RouteSummary) (valueobject.RouteSummary, error) {
	var (
		gasPrice *big.Float
	)

	amountIn, ok := new(big.Int).SetString(params.AmountIn, 10)
	if !ok {
		return valueobject.RouteSummary{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid routeSummary.amountIn [%s]",
			params.AmountIn,
		)
	}

	amountOut, ok := new(big.Int).SetString(params.AmountOut, 10)
	if !ok {
		return valueobject.RouteSummary{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid routeSummary.amountOut [%s]",
			params.AmountOut,
		)
	}

	if len(params.GasPrice) > 0 {
		gasPrice, ok = new(big.Float).SetString(params.GasPrice)
		if !ok {
			return valueobject.RouteSummary{}, errors.Wrapf(
				ErrInvalidRoute,
				"invalid routeSummary.gasPrice [%s]",
				params.GasPrice,
			)
		}
	}

	amountInUSD, err := strconv.ParseFloat(params.AmountInUSD, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid routeSummary.amountInUsd [%s]",
			params.AmountInUSD,
		)
	}

	amountOutUSD, err := strconv.ParseFloat(params.AmountOutUSD, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid routeSummary.amountOutUsd [%s]",
			params.AmountOutUSD,
		)
	}

	gas, err := strconv.ParseInt(params.Gas, 10, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid routerouteSummary.gas [%s]",
			params.Gas,
		)
	}

	gasUSD, err := strconv.ParseFloat(params.GasUSD, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid routerouteSummary.gasUsd [%s]",
			params.GasUSD,
		)
	}

	extraFee, err := transformExtraFeeParams(params.ExtraFee)
	if err != nil {
		return valueobject.RouteSummary{}, err
	}

	if extraFee.IsChargeFeeByCurrencyIn() {
		actualFeeAmount := extraFee.CalcActualFeeAmount(amountIn)

		if actualFeeAmount.Cmp(amountIn) > 0 {
			return valueobject.RouteSummary{}, errors.Wrapf(
				ErrFeeAmountGreaterThanAmountIn,
				"feeAmount: [%s], amountIn: [%s]",
				actualFeeAmount.String(),
				amountIn.String(),
			)
		}
	}

	if extraFee.IsChargeFeeByCurrencyOut() {
		actualFeeAmount := extraFee.CalcActualFeeAmount(amountOut)

		if actualFeeAmount.Cmp(amountOut) > 0 {
			return valueobject.RouteSummary{}, errors.Wrapf(
				ErrFeeAmountGreaterThanAmountOut,
				"feeAmount: [%s], amountOut: [%s]",
				actualFeeAmount.String(),
				amountOut.String(),
			)
		}
	}

	route := make([][]valueobject.Swap, 0, len(params.Route))
	for _, pathParams := range params.Route {
		path := make([]valueobject.Swap, 0, len(pathParams))
		for _, swapParams := range pathParams {
			swap, err := transformSwapParams(swapParams)
			if err != nil {
				return valueobject.RouteSummary{}, err
			}

			path = append(path, swap)
		}

		route = append(route, path)
	}

	return valueobject.RouteSummary{
		TokenIn:                     params.TokenIn,
		AmountIn:                    amountIn,
		AmountInUSD:                 amountInUSD,
		TokenInMarketPriceAvailable: params.TokenInMarketPriceAvailable,

		TokenOut:                     params.TokenOut,
		AmountOut:                    amountOut,
		AmountOutUSD:                 amountOutUSD,
		TokenOutMarketPriceAvailable: params.TokenOutMarketPriceAvailable,

		Gas:      gas,
		GasPrice: gasPrice,
		GasUSD:   gasUSD,

		ExtraFee: extraFee,

		Route: route,
	}, nil
}

// transformExtraFeeParams transforms params.ExtraFee to valueobject.ExtraFee
func transformExtraFeeParams(params params.ExtraFee) (valueobject.ExtraFee, error) {
	if params.FeeAmount == "0" {
		return valueobject.ZeroExtraFee, nil
	}

	feeAmount, ok := new(big.Int).SetString(params.FeeAmount, 10)
	if !ok {
		return valueobject.ExtraFee{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid routeSummary.extraFee.feeAmount [%s]",
			params.FeeAmount,
		)
	}

	return valueobject.ExtraFee{
		FeeAmount:   feeAmount,
		ChargeFeeBy: valueobject.ChargeFeeBy(params.ChargeFeeBy),
		IsInBps:     params.IsInBps,
		FeeReceiver: params.FeeReceiver,
	}, nil
}

// transformSwapParams transforms params.Swap to valueobject.Swap
func transformSwapParams(params params.Swap) (valueobject.Swap, error) {
	limitReturnAmount, ok := new(big.Int).SetString(params.LimitReturnAmount, 10)
	if !ok {
		return valueobject.Swap{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid swap.limitReturnAmount [%s]",
			params.LimitReturnAmount,
		)
	}

	swapAmount, ok := new(big.Int).SetString(params.SwapAmount, 10)
	if !ok {
		return valueobject.Swap{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid swap.SwapAmount [%s]",
			params.SwapAmount,
		)
	}

	amountOut, ok := new(big.Int).SetString(params.AmountOut, 10)
	if !ok {
		return valueobject.Swap{}, errors.Wrapf(
			ErrInvalidRoute,
			"invalid swap.AmountOut [%s]",
			params.AmountOut,
		)
	}

	return valueobject.Swap{
		Pool:              params.Pool,
		TokenIn:           params.TokenIn,
		TokenOut:          params.TokenOut,
		LimitReturnAmount: limitReturnAmount,
		SwapAmount:        swapAmount,
		AmountOut:         amountOut,
		Exchange:          valueobject.Exchange(params.Exchange),
		PoolLength:        params.PoolLength,
		PoolType:          params.PoolType,
		PoolExtra:         params.PoolExtra,
		Extra:             params.Extra,
	}, nil
}
