package api

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getrouteencode"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GetRoutes [GET /routes] Find best routes
func GetRoutes(
	validator IGetRoutesParamsValidator,
	useCase IGetRoutesUseCase,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromGinContext(ginCtx, "GetRoutes")
		defer span.End()

		span.SetTag("request-uri", ginCtx.Request.URL.RequestURI())

		var queryParams params.GetRoutesParams
		if err := ginCtx.ShouldBindQuery(&queryParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.WithMessagef(ErrBindQueryFailed, "[GetRoutes] err: [%v]", err),
			)
			return
		}
		queryParams.ClientId = clientid.ExtractClientID(ginCtx)

		if err := validator.Validate(queryParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		query, err := transformGetRoutesParams(ginCtx, queryParams)
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

func transformGetRoutesParams(ginCtx *gin.Context, params params.GetRoutesParams) (dto.GetRoutesQuery, error) {
	amountIn, ok := new(big.Int).SetString(params.AmountIn, 10)
	if !ok {
		return dto.GetRoutesQuery{}, errors.WithMessagef(
			ErrInvalidValue,
			"amountIn: [%s]",
			params.AmountIn,
		)
	}

	var gasPrice *big.Float
	if params.GasPrice != "" {
		gasPrice, ok = new(big.Float).SetString(params.GasPrice)
		if !ok {
			return dto.GetRoutesQuery{}, errors.WithMessagef(
				ErrInvalidValue,
				"gasPrice: [%s]",
				params.GasPrice,
			)
		}
	}

	extraFee := valueobject.ZeroExtraFee
	if params.FeeAmount != "" {
		feeAmounts := utils.TransformSliceParams(params.FeeAmount)
		feeReceivers := utils.TransformSliceParams(params.FeeReceiver)
		if len(feeReceivers) != len(feeAmounts) {
			return dto.GetRoutesQuery{}, errors.WithMessagef(
				ErrInvalidValue,
				"feeReceivers: [%s], feeAmounts: [%s]",
				params.FeeReceiver,
				params.FeeAmount,
			)
		}

		var err error
		feeAmountsBigInt := lo.Map(feeAmounts, func(item string, index int) *big.Int {
			feeAmount, ok := new(big.Int).SetString(item, 10)
			if !ok {
				err = validator.NewValidationError("feeAmount", "invalid")
				return nil
			}
			return feeAmount
		})
		if err != nil {
			return dto.GetRoutesQuery{}, err
		}

		for _, feeReceiver := range feeReceivers {
			if !validator.IsEthereumAddress(feeReceiver) {
				return dto.GetRoutesQuery{}, err
			}
		}

		extraFee = valueobject.ExtraFee{
			FeeAmount:   feeAmountsBigInt,
			ChargeFeeBy: valueobject.ChargeFeeBy(params.ChargeFeeBy),
			IsInBps:     params.IsInBps,
			FeeReceiver: feeReceivers,
		}

		actualFeeAmount := extraFee.CalcActualFeeAmount(amountIn)

		if extraFee.IsChargeFeeByCurrencyIn() && actualFeeAmount.Cmp(amountIn) > 0 {
			return dto.GetRoutesQuery{}, errors.WithMessagef(
				ErrFeeAmountGreaterThanAmountIn,
				"feeAmount: [%s], amountIn: [%s]",
				actualFeeAmount.String(),
				amountIn.String(),
			)
		}
	}

	if params.Index != "" {
		validIndex := valueobject.IndexType(params.Index)
		if validIndex != valueobject.Composite && validIndex != valueobject.NativeTvl {
			return dto.GetRoutesQuery{}, errors.WithMessagef(
				ErrInvalidValue,
				"index: [%s]",
				params.Index,
			)
		}
	}

	excludedSources := utils.TransformSliceParams(params.ExcludedSources)
	if params.ExcludeRFQSources {
		excludedSources = append(excludedSources,
			getrouteencode.GetExcludedRFQSources()...)
	}
	return dto.GetRoutesQuery{
		TokenIn:             utils.CleanUpParam(params.TokenIn),
		TokenOut:            utils.CleanUpParam(params.TokenOut),
		AmountIn:            amountIn,
		IncludedSources:     utils.TransformSliceParams(params.IncludedSources),
		ExcludedSources:     excludedSources,
		OnlyScalableSources: params.OnlyScalableSources,
		OnlySinglePath:      params.OnlySinglePath,
		GasInclude:          params.GasInclude,
		GasPrice:            gasPrice,
		ExtraFee:            extraFee,
		ExcludedPools:       mapset.NewThreadUnsafeSet(utils.TransformSliceParams(params.ExcludedPools)...),
		ClientId:            params.ClientId,
		BotScore:            requestid.ExtractBotScore(ginCtx),
		Index:               params.Index,
		PoolIds:             utils.TransformSliceParams(params.PoolIds),
	}, nil
}

func transformGetRoutesResult(result *dto.GetRoutesResult) *params.GetRoutesResponse {
	if result == nil {
		return nil
	}

	summary := transformRouteSummary(result.RouteSummary)
	summary.RouteID = result.RouteSummary.RouteID
	summary.Checksum = fmt.Sprintf("%d", result.Checksum)
	summary.Timestamp = result.RouteSummary.Timestamp

	return &params.GetRoutesResponse{
		RouteSummary:  summary,
		RouterAddress: result.RouterAddress,
	}
}

func transformRouteSummary(routeSummary *valueobject.RouteSummary) *params.RouteSummary {
	if routeSummary == nil {
		return nil
	}
	var alphaFee *params.AlphaFeeV2
	if routeSummary.AlphaFee != nil {
		alphaFee = &params.AlphaFeeV2{
			AMMAmount: routeSummary.AlphaFee.AMMAmount.String(),
			SwapReductions: lo.Map(routeSummary.AlphaFee.SwapReductions, func(
				swapReduction entity.AlphaFeeV2SwapReduction,
				_ int,
			) params.AlphaFeeV2SwapReduction {
				return params.AlphaFeeV2SwapReduction{
					ExecutedId:      swapReduction.ExecutedId,
					Token:           swapReduction.TokenOut,
					ReduceAmount:    swapReduction.ReduceAmount.String(),
					ReduceAmountUsd: swapReduction.ReduceAmountUsd,
				}
			}),
		}
	}
	return &params.RouteSummary{
		TokenIn:     routeSummary.TokenIn,
		AmountIn:    routeSummary.AmountIn.String(),
		AmountInUSD: strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),

		TokenOut:     routeSummary.TokenOut,
		AmountOut:    routeSummary.AmountOut.String(),
		AmountOutUSD: strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),

		Gas:      strconv.FormatInt(routeSummary.Gas, 10),
		GasPrice: routeSummary.GasPrice.Text('f', -1),
		GasUSD:   strconv.FormatFloat(routeSummary.GasUSD, 'f', -1, 64),
		L1FeeUSD: strconv.FormatFloat(routeSummary.L1FeeUSD, 'f', -1, 64),
		ExtraFee: transformExtraFee(routeSummary.ExtraFee),
		Route:    transformRoute(routeSummary.Route),
		AlphaFee: alphaFee,
	}
}

func transformExtraFee(extraFee valueobject.ExtraFee) params.ExtraFee {
	return params.ExtraFee{
		FeeAmount: strings.Join(lo.Map(extraFee.FeeAmount, func(item *big.Int, index int) string {
			return item.String()
		}), ","),
		ChargeFeeBy: string(extraFee.ChargeFeeBy),
		FeeReceiver: strings.Join(extraFee.FeeReceiver, ","),
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
		Pool:       swap.Pool,
		TokenIn:    swap.TokenIn,
		TokenOut:   swap.TokenOut,
		SwapAmount: swap.SwapAmount.String(),
		AmountOut:  swap.AmountOut.String(),
		Exchange:   string(swap.Exchange),
		PoolType:   swap.PoolType,
		PoolExtra:  swap.PoolExtra,
		Extra:      swap.Extra,
	}
}
