package getroute

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/crypto"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

type useCase struct {
	aggregator IAggregator

	tokenRepository    ITokenRepository
	gasRepository      IGasRepository
	alphaFeeRepository IAlphaFeeRepository
	l1FeeEstimator     IL1FeeEstimator

	onchainpriceRepository IOnchainPriceRepository

	config Config
	mu     sync.RWMutex
}

func NewUseCase(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	onchainpriceRepository IOnchainPriceRepository,
	routeCacheRepository IRouteCacheRepository,
	gasRepository IGasRepository,
	alphaFeeRepository IAlphaFeeRepository,
	l1FeeEstimator IL1FeeEstimator,
	poolManager IPoolManager,
	finderEngine finderEngine.IPathFinderEngine,
	config Config,
) *useCase {
	aggregator := NewAggregator(
		poolRankRepository,
		tokenRepository,
		onchainpriceRepository,
		poolManager,
		config.Aggregator,
		finderEngine,
	)
	correlatedPairsAggregator := NewCorrelatedPairs(
		aggregator,
		poolRankRepository,
		tokenRepository,
		onchainpriceRepository,
		poolManager,
		config,
	)

	var finalizedAggregator IAggregator
	if config.Aggregator.FeatureFlags.IsRouteCachedEnable {
		aggregatorWithCache := NewCache(correlatedPairsAggregator, routeCacheRepository, poolManager, config.Cache,
			finderEngine, tokenRepository, onchainpriceRepository)
		finalizedAggregator = aggregatorWithCache
	} else {
		finalizedAggregator = correlatedPairsAggregator
	}
	aggregatorWithChargeExtraFee := NewChargeExtraFee(finalizedAggregator)

	return &useCase{
		aggregator:             aggregatorWithChargeExtraFee,
		tokenRepository:        tokenRepository,
		gasRepository:          gasRepository,
		l1FeeEstimator:         l1FeeEstimator,
		config:                 config,
		alphaFeeRepository:     alphaFeeRepository,
		onchainpriceRepository: onchainpriceRepository,
	}
}

func (u *useCase) Handle(ctx context.Context, query dto.GetRoutesQuery) (*dto.GetRoutesResult, error) {
	if env.StringFromEnv(envvar.CalcAmountOutCounterMetricEnabled, "") != "" {
		calcAmountOutCounter := metrics.NewCalcAmountOutCounter()
		ctx = context.WithValue(ctx, metrics.CalcAmountOutCounterContextKey, calcAmountOutCounter)
		defer calcAmountOutCounter.CommitMetrics(ctx)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] useCase.Handle")
	defer span.End()

	originalTokenIn := query.TokenIn
	originalTokenOut := query.TokenOut

	wrappedTokensQuery, err := u.wrapTokens(query)
	if err != nil {
		return nil, err
	}

	params, err := u.getAggregateParams(ctx, wrappedTokensQuery)
	if err != nil {
		return nil, err
	}

	params.AmountInUsd = utils.CalcTokenAmountUsd(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD)
	if params.AmountInUsd > MaxAmountInUSD {
		return nil, ErrAmountInIsGreaterThanMaxAllowed
	}

	routeSummaries, err := u.aggregator.Aggregate(ctx, params)
	if err != nil {
		return nil, err
	}
	routeSummary := routeSummaries.GetBestRouteSummary()

	routeID := requestid.GetRequestIDFromCtx(ctx)

	// Only save route which including alphaFee
	if routeSummary.AlphaFee != nil {
		err = u.alphaFeeRepository.Save(ctx, routeID, routeSummary.AlphaFee)
		if err != nil {
			return nil, err
		}
	}

	routeSummary.TokenIn = originalTokenIn
	routeSummary.TokenOut = originalTokenOut
	routeSummary.Timestamp = time.Now().Unix()
	routeSummary.RouteID = routeID

	checksum := crypto.NewChecksum(routeSummary, u.config.Salt)

	// TOTO: this line of code will be removed later, do not return alpha
	if !u.config.Aggregator.FeatureFlags.ShouldReturnAlphaFee {
		routeSummary.AlphaFee = nil
	}

	return &dto.GetRoutesResult{
		RouteSummary:  routeSummary,
		Checksum:      checksum.Hash(),
		RouterAddress: u.config.RouterAddress,
	}, nil
}

func (u *useCase) ApplyConfig(config Config) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.config = config
	if u.aggregator != nil {
		u.aggregator.ApplyConfig(config)
	}
}

// wrapTokens wraps tokens in query and returns the query
func (u *useCase) wrapTokens(query dto.GetRoutesQuery) (dto.GetRoutesQuery, error) {
	wrappedTokenIn, err := eth.ConvertEtherToWETH(query.TokenIn, u.config.ChainID)
	if err != nil {
		return dto.GetRoutesQuery{}, err
	}

	wrappedTokenOut, err := eth.ConvertEtherToWETH(query.TokenOut, u.config.ChainID)
	if err != nil {
		return dto.GetRoutesQuery{}, err
	}

	query.TokenIn = wrappedTokenIn
	query.TokenOut = wrappedTokenOut

	return query, nil
}

// TODO: remove unnecessary get token and price here, we need to re-fetch tokens and prices for alpha fee calculation
func (u *useCase) getAggregateParams(ctx context.Context, query dto.GetRoutesQuery) (*types.AggregateParams, error) {
	tokenByAddress, err := u.getTokenByAddress(ctx, query.TokenIn, query.TokenOut, u.config.GasTokenAddress)
	if err != nil {
		return nil, err
	}

	tokenIn, ok := tokenByAddress[query.TokenIn]
	if !ok {
		return nil, errors.WithMessagef(ErrTokenNotFound, "tokenIn: [%s]", query.TokenIn)
	}

	tokenOut, ok := tokenByAddress[query.TokenOut]
	if !ok {
		return nil, errors.WithMessagef(ErrTokenNotFound, "tokenOut: [%s]", query.TokenOut)
	}

	tokenInPriceUSD, tokenOutPriceUSD, gasTokenPriceUSD, err := u.getTokensPriceUSD(ctx, query.TokenIn, query.TokenOut,
		u.config.GasTokenAddress)
	if err != nil {
		return nil, err
	}

	gasPrice, err := u.getGasPrice(ctx, query.GasPrice)
	if err != nil {
		return nil, err
	}

	var l1FeeOverhead, l1FeePerPool *big.Int
	if valueobject.IsL1FeeEstimateSupported(u.config.ChainID) {
		if l1FeeOverhead, l1FeePerPool, err = u.l1FeeEstimator.EstimateL1Fees(ctx); err != nil {
			klog.Errorf(ctx, "failed to estimate l1 fees: %v", err)
		}
	}

	sources := u.getSources(query.ClientId, query.IncludedSources, query.ExcludedSources, query.OnlyScalableSources)

	index := valueobject.NativeTvl
	if u.config.Aggregator.FeatureFlags.IsLiquidityScoreIndexEnable {
		if query.Index != "" {
			index = valueobject.IndexType(query.Index)
		} else {
			index = valueobject.IndexType(u.config.DefaultPoolsIndex)
		}
	}

	var kyberLimitOrderAllowedSenders string
	if u.config.Aggregator.FeatureFlags.IsKyberPrivateLimitOrdersEnabled && query.ClientId == clientid.KyberSwap {
		kyberLimitOrderAllowedSenders = u.config.KyberExecutorAddress
	}

	return &types.AggregateParams{
		TokenIn:                       tokenIn,
		TokenOut:                      tokenOut,
		GasToken:                      tokenByAddress[u.config.GasTokenAddress],
		TokenInPriceUSD:               tokenInPriceUSD,
		TokenOutPriceUSD:              tokenOutPriceUSD,
		GasTokenPriceUSD:              gasTokenPriceUSD,
		AmountIn:                      query.AmountIn,
		Sources:                       sources,
		SaveGas:                       query.SaveGas,
		OnlySinglePath:                query.OnlySinglePath,
		GasInclude:                    query.GasInclude,
		GasPrice:                      gasPrice,
		L1FeeOverhead:                 l1FeeOverhead,
		L1FeePerPool:                  l1FeePerPool,
		ExtraFee:                      query.ExtraFee,
		IsHillClimbEnabled:            u.config.Aggregator.FeatureFlags.IsHillClimbEnabled,
		Index:                         index,
		ExcludedPools:                 query.ExcludedPools,
		ClientId:                      query.ClientId,
		KyberLimitOrderAllowedSenders: kyberLimitOrderAllowedSenders,
		IsScaleHelperClient:           lo.Contains(u.config.ScaleHelperClients, query.ClientId),
		EnableAlphaFee:                u.config.Aggregator.FeatureFlags.IsAlphaFeeReductionEnable,
		EnableHillClaimForAlphaFee:    u.config.Aggregator.FeatureFlags.IsHillClimbEnabledForAMMBestRoute,
	}, nil
}

func (u *useCase) getTokenByAddress(ctx context.Context, addresses ...string) (map[string]entity.Token, error) {
	tokens, err := u.tokenRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = *token
	}

	return tokenByAddress, nil
}

func (u *useCase) getTokensPriceUSD(ctx context.Context, tokenIn, tokenOut, gasToken string) (float64, float64, float64,
	error) {
	priceByAddress, err := u.onchainpriceRepository.FindByAddresses(ctx, []string{tokenIn, tokenOut, gasToken})
	if err != nil {
		return 0, 0, 0, err
	}

	// use sell price for token in
	tokenInPriceUSD := 0.0
	if price, ok := priceByAddress[tokenIn]; ok && price != nil && price.USDPrice.Sell != nil {
		tokenInPriceUSD, _ = price.USDPrice.Sell.Float64()
	}

	// use buy price for token out and gas
	tokenOutPriceUSD := 0.0
	if price, ok := priceByAddress[tokenOut]; ok && price != nil && price.USDPrice.Buy != nil {
		tokenOutPriceUSD, _ = price.USDPrice.Buy.Float64()
	}
	gasTokenPriceUSD := 0.0
	if price, ok := priceByAddress[gasToken]; ok && price != nil && price.USDPrice.Buy != nil {
		gasTokenPriceUSD, _ = price.USDPrice.Buy.Float64()
	}

	return tokenInPriceUSD, tokenOutPriceUSD, gasTokenPriceUSD, nil
}

func (u *useCase) getGasPrice(ctx context.Context, customGasPrice *big.Float) (*big.Float, error) {
	if customGasPrice != nil {
		return customGasPrice, nil
	}

	suggestedGasPrice, err := u.gasRepository.GetSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	return new(big.Float).SetInt(suggestedGasPrice), nil
}

func (u *useCase) getSources(clientId string, includedSources []string, excludedSources []string,
	onlyScalableSources bool) []string {
	var sources mapset.Set[string]
	if len(includedSources) > 0 {
		sources = mapset.NewThreadUnsafeSet(includedSources...)
	} else {
		sources = mapset.NewThreadUnsafeSet(u.config.AvailableSources...)
	}

	sources.RemoveAll(excludedSources...)

	if excludedSourcesByClient, ok := u.config.ExcludedSourcesByClient[clientId]; ok {
		sources.RemoveAll(excludedSourcesByClient...)
	}

	if onlyScalableSources {
		sources.RemoveAll(u.config.UnscalableSources...)
	}

	return sources.ToSlice()
}
