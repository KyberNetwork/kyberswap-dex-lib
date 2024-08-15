package api

import (
	"math/big"
	"net/http"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getrouteencode"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GetRouteEncode [GET /route/encode] Find best route with encode
func GetRouteEncode(
	validator IGetRouteEncodeParamsValidator,
	getRoutesUseCase IGetRoutesUseCase,
	buildRoutesUseCase IBuildRouteUseCase,
	getTokensUseCase IGetTokensUseCase,
	nowFunc func() time.Time,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromGinContext(ginCtx, "GetRouteEncode")
		defer span.End()

		span.SetTag("request-uri", ginCtx.Request.URL.RequestURI())

		clientIDFromHeader := clientid.ExtractClientID(ginCtx)

		var queryParams params.GetRouteEncodeParams
		if err := ginCtx.ShouldBindQuery(&queryParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.WithMessagef(ErrBindQueryFailed, "[GetRouteEncode] err: [%v]", err),
			)
			return
		}

		if err := validator.Validate(ctx, queryParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		// if source param is empty, use clientID from header as the source
		if queryParams.ClientData.Source == "" {
			queryParams.ClientData = params.ClientData{
				Source: clientIDFromHeader,
			}
		}

		getRoutesQuery, err := transformFromGetRouteEncodeToGetRoutesQuery(queryParams)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		getRoutesResult, err := getRoutesUseCase.Handle(ctx, getRoutesQuery)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		if getRoutesResult.RouteSummary == nil {
			RespondFailure(ginCtx, ErrRouteNotFound)
			return
		}

		buildRouteCommand, err := buildBuildRouteCommand(queryParams, getRoutesResult, nowFunc)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		buildRouteResult, err := buildRoutesUseCase.Handle(ctx, buildRouteCommand)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		getTokensQuery := buildGetTokensQuery(getRoutesResult)

		getTokensResult, err := getTokensUseCase.Handle(ctx, getTokensQuery)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		response, err := buildGetRouteEncodeResponse(getRoutesResult, buildRouteResult, getTokensResult)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		ginCtx.AbortWithStatusJSON(http.StatusOK, response)
	}
}

// transformFromGetRouteEncodeToGetRoutesQuery
// inherit and edit from transformGetRoutesParams https://github.com/KyberNetwork/router-service/blob/cc2fc3f65aa62f648dd4cf9244cb3d247042b1b4/internal/pkg/api/get_routes.go#L59
func transformFromGetRouteEncodeToGetRoutesQuery(params params.GetRouteEncodeParams) (dto.GetRoutesQuery, error) {
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
		feeAmount, ok := new(big.Int).SetString(params.FeeAmount, 10)
		if !ok {
			return dto.GetRoutesQuery{}, errors.WithMessagef(
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
			return dto.GetRoutesQuery{}, errors.WithMessagef(
				ErrFeeAmountGreaterThanAmountIn,
				"feeAmount: [%s], amountIn: [%s]",
				actualFeeAmount.String(),
				amountIn.String(),
			)
		}
	}

	return dto.GetRoutesQuery{
		TokenIn:         utils.CleanUpParam(params.TokenIn),
		TokenOut:        utils.CleanUpParam(params.TokenOut),
		AmountIn:        amountIn,
		IncludedSources: utils.TransformSliceParams(params.Dexes),
		ExcludedSources: getrouteencode.GetExcludedSources(),
		SaveGas:         params.SaveGas,
		GasInclude:      params.GasInclude,
		GasPrice:        gasPrice,
		ExtraFee:        extraFee,
		ExcludedPools:   mapset.NewThreadUnsafeSet[string](),
		ClientId:        params.ClientData.Source,
	}, nil
}

func buildBuildRouteCommand(
	params params.GetRouteEncodeParams,
	getRoutesResult *dto.GetRoutesResult,
	nowFunc func() time.Time,
) (dto.BuildRouteCommand, error) {
	deadline := params.Deadline
	if params.Deadline == 0 {
		deadline = nowFunc().Add(valueobject.DefaultDeadline).Unix()
	}

	return dto.BuildRouteCommand{
		RouteSummary:      *getRoutesResult.RouteSummary,
		Recipient:         utils.CleanUpParam(params.To),
		Deadline:          deadline,
		SlippageTolerance: params.SlippageTolerance,
		Referral:          params.Referral,
		Source:            params.ClientData.Source,
		Permit:            common.FromHex(params.Permit),
	}, nil
}

func buildGetTokensQuery(
	getRoutesResult *dto.GetRoutesResult,
) dto.GetTokensQuery {
	tokenAddressSet := make(map[string]struct{})

	for _, path := range getRoutesResult.RouteSummary.Route {
		for _, swap := range path {
			tokenAddressSet[swap.TokenIn] = struct{}{}
			tokenAddressSet[swap.TokenOut] = struct{}{}
		}
	}

	addresses := make([]string, 0, len(tokenAddressSet))
	for address := range tokenAddressSet {
		addresses = append(addresses, address)
	}

	return dto.GetTokensQuery{
		IDs: addresses,
	}
}

func buildGetRouteEncodeResponse(
	getRoutesResult *dto.GetRoutesResult,
	buildRouteResult *dto.BuildRouteResult,
	getTokensResult *dto.GetTokensResult,
) (params.GetRouteEncodeResponse, error) {

	return params.GetRouteEncodeResponse{
		InputAmount: getRoutesResult.RouteSummary.AmountIn.String(),
		AmountInUsd: getRoutesResult.RouteSummary.AmountInUSD,

		OutputAmount: getRoutesResult.RouteSummary.AmountOut.String(),
		AmountOutUsd: getRoutesResult.RouteSummary.AmountOutUSD,

		TotalGas:     getRoutesResult.RouteSummary.Gas,
		GasPriceGwei: new(big.Float).Quo(getRoutesResult.RouteSummary.GasPrice, big.NewFloat(1e9)).String(),
		GasUsd:       getRoutesResult.RouteSummary.GasUSD,

		ReceivedUsd:     getRoutesResult.RouteSummary.AmountOutUSD - getRoutesResult.RouteSummary.GasUSD,
		Swaps:           transformGetRouteEncodeResponseSwaps(getRoutesResult.RouteSummary.Route),
		Tokens:          transformGetRouteEncodeResponseTokens(getTokensResult.Tokens),
		EncodedSwapData: buildRouteResult.Data,
		RouterAddress:   buildRouteResult.RouterAddress,
	}, nil
}

func transformGetRouteEncodeResponseSwaps(route [][]valueobject.Swap) [][]params.GetRouteEncodeResponseSwap {
	routeParams := make([][]params.GetRouteEncodeResponseSwap, 0, len(route))

	for _, path := range route {
		pathParams := make([]params.GetRouteEncodeResponseSwap, 0, len(path))

		for _, swap := range path {
			pathParams = append(pathParams, transformGetRouteEncodeSwap(swap))
		}

		routeParams = append(routeParams, pathParams)
	}

	return routeParams
}

func transformGetRouteEncodeResponseTokens(tokens []*dto.GetTokensResultToken) map[string]params.GetRouteEncodeResponseToken {
	getRouteEncodeResponseTokenByAddress := make(map[string]params.GetRouteEncodeResponseToken, len(tokens))

	for _, token := range tokens {
		getRouteEncodeResponseTokenByAddress[token.Address] = params.GetRouteEncodeResponseToken{
			Address:  token.Address,
			Symbol:   token.Symbol,
			Name:     token.Name,
			Price:    transformPrice(token.Price),
			Decimals: token.Decimals,
		}
	}

	return getRouteEncodeResponseTokenByAddress
}

func transformGetRouteEncodeSwap(swap valueobject.Swap) params.GetRouteEncodeResponseSwap {
	return params.GetRouteEncodeResponseSwap{
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
		MaxPrice:          "",
	}
}

func transformPrice(price *dto.GetTokensResultPrice) float64 {
	if price == nil {
		return 0
	}

	// We don't always have market price, so it's better to have this fallback
	if price.MarketPrice == 0 {
		return price.Price
	}

	switch price.PreferPriceSource {
	case string(entity.PriceSourceKyberswap):
		return price.Price
	case string(entity.PriceSourceCoingecko):
		return price.MarketPrice
	default:
		return price.MarketPrice
	}
}
