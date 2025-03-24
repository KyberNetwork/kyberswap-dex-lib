package api

import (
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// BuildRoute [POST /route/build] build route
func BuildRoute(validator IBuildRouteParamsValidator, useCase IBuildRouteUseCase, cfg buildroute.Config,
	nowFunc func() time.Time) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromGinContext(ginCtx, "BuildRoute")
		defer span.End()

		clientIDFromHeader := clientid.ExtractClientID(ginCtx)

		var bodyParams params.BuildRouteParams
		if err := ginCtx.ShouldBindJSON(&bodyParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.WithMessagef(
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

		command, err := transformBuildRouteParams(ginCtx, bodyParams, cfg, nowFunc)
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
func transformBuildRouteParams(ginCtx *gin.Context, params params.BuildRouteParams, cfg buildroute.Config,
	nowFunc func() time.Time) (dto.BuildRouteCommand, error) {
	routeSummary, err := transformRouteSummaryParams(params.RouteSummary)
	if err != nil {
		return dto.BuildRouteCommand{}, err
	}

	deadline := params.Deadline
	if params.Deadline == 0 {
		deadline = nowFunc().Add(valueobject.DefaultDeadline).Unix()
	}

	permit := common.FromHex(params.Permit)
	num, _ := strconv.ParseUint(params.RouteSummary.Checksum, 10, 64)

	source := params.Source
	if ginCtx != nil {
		normalizedClientId := strings.ReplaceAll(ginCtx.ClientIP(), ".", "_")
		if forcedSource, ok := cfg.ForceSourceByIp[normalizedClientId]; ok {
			source = forcedSource
		}
	}

	return dto.BuildRouteCommand{
		RouteSummary:        routeSummary,
		Checksum:            num,
		ValidateChecksum:    cfg.ValidateChecksumBySource[source],
		Sender:              params.Sender,
		Recipient:           params.Recipient,
		Permit:              permit,
		Deadline:            deadline,
		SlippageTolerance:   params.SlippageTolerance,
		EnableGasEstimation: params.EnableGasEstimation,
		Source:              source,
		Referral:            params.Referral,
	}, nil
}

// transformRouteSummaryParams transforms params.RouteSummary to valueobject.RouteSummary
func transformRouteSummaryParams(params params.RouteSummary) (valueobject.RouteSummary, error) {
	var (
		gasPrice *big.Float
		l1FeeUSD float64
	)

	amountIn, ok := new(big.Int).SetString(params.AmountIn, 10)
	if !ok {
		return valueobject.RouteSummary{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid routeSummary.amountIn [%s]",
			params.AmountIn,
		)
	}

	amountOut, ok := new(big.Int).SetString(params.AmountOut, 10)
	if !ok {
		return valueobject.RouteSummary{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid routeSummary.amountOut [%s]",
			params.AmountOut,
		)
	}

	if len(params.GasPrice) > 0 {
		gasPrice, ok = new(big.Float).SetString(params.GasPrice)
		if !ok {
			return valueobject.RouteSummary{}, errors.WithMessagef(
				ErrInvalidRoute,
				"invalid routeSummary.gasPrice [%s]",
				params.GasPrice,
			)
		}
	}

	amountInUSD, err := strconv.ParseFloat(params.AmountInUSD, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid routeSummary.amountInUsd [%s]",
			params.AmountInUSD,
		)
	}

	amountOutUSD, err := strconv.ParseFloat(params.AmountOutUSD, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid routeSummary.amountOutUsd [%s]",
			params.AmountOutUSD,
		)
	}

	gas, err := strconv.ParseInt(params.Gas, 10, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid routeSummary.gas [%s]",
			params.Gas,
		)
	}

	gasUSD, err := strconv.ParseFloat(params.GasUSD, 64)
	if err != nil {
		return valueobject.RouteSummary{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid routeSummary.gasUsd [%s]",
			params.GasUSD,
		)
	}

	if len(params.L1FeeUSD) > 0 {
		l1FeeUSD, err = strconv.ParseFloat(params.L1FeeUSD, 64)
		if err != nil {
			return valueobject.RouteSummary{}, errors.WithMessagef(
				ErrInvalidRoute,
				"invalid routeSummary.l1FeeUsd [%s]",
				params.L1FeeUSD,
			)
		}
	}

	extraFee, err := transformExtraFeeParams(params.ExtraFee)
	if err != nil {
		return valueobject.RouteSummary{}, err
	}

	if extraFee.IsChargeFeeByCurrencyIn() {
		actualFeeAmount := extraFee.CalcActualFeeAmount(amountIn)

		if actualFeeAmount.Cmp(amountIn) > 0 {
			return valueobject.RouteSummary{}, errors.WithMessagef(
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
			return valueobject.RouteSummary{}, errors.WithMessagef(
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
		L1FeeUSD: l1FeeUSD,

		ExtraFee:  extraFee,
		Timestamp: params.Timestamp,

		Route:   route,
		RouteID: params.RouteID,
	}, nil
}

// transformExtraFeeParams transforms params.ExtraFee to valueobject.ExtraFee
func transformExtraFeeParams(params params.ExtraFee) (valueobject.ExtraFee, error) {
	if params.FeeAmount == "0" {
		return valueobject.ZeroExtraFee, nil
	}

	feeAmount, ok := new(big.Int).SetString(params.FeeAmount, 10)
	if !ok {
		return valueobject.ExtraFee{}, errors.WithMessagef(
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
		return valueobject.Swap{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid swap.limitReturnAmount [%s]",
			params.LimitReturnAmount,
		)
	}

	swapAmount, ok := new(big.Int).SetString(params.SwapAmount, 10)
	if !ok {
		return valueobject.Swap{}, errors.WithMessagef(
			ErrInvalidRoute,
			"invalid swap.SwapAmount [%s]",
			params.SwapAmount,
		)
	}

	amountOut, ok := new(big.Int).SetString(params.AmountOut, 10)
	if !ok {
		return valueobject.Swap{}, errors.WithMessagef(
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
