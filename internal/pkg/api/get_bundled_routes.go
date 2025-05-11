package api

import (
	"math/big"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GetBundledRoutes [GET/POST /bundled-routes] Find best results for multiple routes consecutively
func GetBundledRoutes(
	validator IGetBundledRoutesParamsValidator,
	useCase IGetBundledRoutesUseCase,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		span, ctx := tracer.StartSpanFromGinContext(ginCtx, "GetBundledRoutes")
		defer span.End()

		span.SetTag("request-uri", ginCtx.Request.URL.RequestURI())

		var queryParams params.GetBundledRoutesParams
		if err := ginCtx.ShouldBind(&queryParams); err != nil {
			RespondFailure(
				ginCtx,
				errors.WithMessagef(ErrBindQueryFailed, "[GetBundledRoutes] err: [%v]", err),
			)
			return
		}
		queryParams.ClientId = clientid.ExtractClientID(ginCtx)

		if err := validator.ValidateBundled(queryParams); err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		query, err := transformGetBundledRoutesParams(ginCtx, queryParams)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		result, err := useCase.Handle(ctx, query)
		if err != nil {
			RespondFailure(ginCtx, err)
			return
		}

		response := transformGetBundledRoutesResult(result)

		RespondSuccess(ginCtx, response)
	}
}

func transformGetBundledRoutesParams(ginCtx *gin.Context, params params.GetBundledRoutesParams) (dto.GetBundledRoutesQuery, error) {
	pairs := make([]*dto.GetBundledRoutesQueryPair, 0, len(params.TokensIn))
	for i, tokenIn := range params.TokensIn {
		amountIn, ok := new(big.Int).SetString(params.AmountsIn[i], 10)
		if !ok {
			return dto.GetBundledRoutesQuery{}, errors.WithMessagef(
				ErrInvalidValue,
				"amountIn: [%s]",
				params.AmountsIn[i],
			)
		}
		pairs = append(pairs, &dto.GetBundledRoutesQueryPair{
			TokenIn:  utils.CleanUpParam(tokenIn),
			TokenOut: utils.CleanUpParam(params.TokensOut[i]),
			AmountIn: amountIn,
		})
	}

	var gasPrice *big.Float
	if params.GasPrice != "" {
		var ok bool
		gasPrice, ok = new(big.Float).SetString(params.GasPrice)
		if !ok {
			return dto.GetBundledRoutesQuery{}, errors.WithMessagef(
				ErrInvalidValue,
				"gasPrice: [%s]",
				params.GasPrice,
			)
		}
	}

	if params.Index != "" && params.Index != string(valueobject.Composite) && params.Index != string(valueobject.LiquidityScore) && params.Index != string(valueobject.NativeTvl) {
		return dto.GetBundledRoutesQuery{}, errors.WithMessagef(
			ErrInvalidValue,
			"index: [%s]",
			params.Index,
		)
	}

	return dto.GetBundledRoutesQuery{
		Pairs:                  pairs,
		IncludedSources:        utils.TransformSliceParams(params.IncludedSources),
		ExcludedSources:        utils.TransformSliceParams(params.ExcludedSources),
		OnlyScalableSources:    params.OnlyScalableSources,
		GasInclude:             params.GasInclude,
		GasPrice:               gasPrice,
		ExcludedPools:          mapset.NewThreadUnsafeSet(utils.TransformSliceParams(params.ExcludedPools)...),
		OverridePools:          params.OverridePools,
		ExtraWhitelistedTokens: utils.TransformSliceParams(params.ExtraWhitelistedTokens),
		ClientId:               params.ClientId,
		BotScore:               requestid.ExtractBotScore(ginCtx),
		Index:                  params.Index,
	}, nil
}

func transformGetBundledRoutesResult(result *dto.GetBundledRoutesResult) *params.GetBundledRoutesResponse {
	if result == nil {
		return nil
	}

	return &params.GetBundledRoutesResponse{
		RoutesSummary: lo.Map(result.RoutesSummary,
			func(s *valueobject.RouteSummary, _ int) *params.RouteSummary { return transformRouteSummary(s) }),
		RouterAddress: result.RouterAddress,
	}
}
