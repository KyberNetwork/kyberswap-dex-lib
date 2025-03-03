package getcustomroute

import (
	"context"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type useCase struct {
	aggregator             IAggregator
	tokenRepository        ITokenRepository
	gasRepository          IGasRepository
	l1FeeEstimator         IL1FeeEstimator
	onchainpriceRepository IOnchainPriceRepository

	config Config
	mu     sync.Mutex
}

func NewCustomRoutesUseCase(
	poolFactory IPoolFactory,
	tokenRepository ITokenRepository,
	onchainpriceRepository IOnchainPriceRepository,
	gasRepository IGasRepository,
	l1FeeEstimator IL1FeeEstimator,
	poolManager IPoolManager,
	poolRepository IPoolRepository,
	finderEngine finderEngine.IPathFinderEngine,
	config Config,
) *useCase {
	aggregator := NewCustomAggregator(
		poolFactory,
		tokenRepository,
		onchainpriceRepository,
		poolManager,
		poolRepository,
		config.Aggregator,
		finderEngine,
	)

	return &useCase{
		aggregator:             aggregator,
		tokenRepository:        tokenRepository,
		gasRepository:          gasRepository,
		l1FeeEstimator:         l1FeeEstimator,
		onchainpriceRepository: onchainpriceRepository,

		config: config,
	}
}

func (u *useCase) Handle(ctx context.Context, query dto.GetCustomRoutesQuery) (*dto.GetRoutesResult, error) {
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

	amountInUSD := utils.CalcTokenAmountUsd(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD)
	if amountInUSD > getroute.MaxAmountInUSD {
		return nil, getroute.ErrAmountInIsGreaterThanMaxAllowed
	}

	routeSummary, err := u.aggregator.Aggregate(ctx, params, query.PoolIds)
	if err != nil {
		return nil, err
	}

	routeSummary.TokenIn = originalTokenIn
	routeSummary.TokenOut = originalTokenOut

	return &dto.GetRoutesResult{
		RouteSummary:  routeSummary,
		RouterAddress: u.config.RouterAddress,
	}, nil
}

func (u *useCase) ApplyConfig(config Config) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.config = config
}

// wrapTokens wraps tokens in query and returns the query
func (u *useCase) wrapTokens(query dto.GetCustomRoutesQuery) (dto.GetCustomRoutesQuery, error) {
	wrappedTokenIn, err := eth.ConvertEtherToWETH(query.TokenIn, u.config.ChainID)
	if err != nil {
		return dto.GetCustomRoutesQuery{}, err
	}

	wrappedTokenOut, err := eth.ConvertEtherToWETH(query.TokenOut, u.config.ChainID)
	if err != nil {
		return dto.GetCustomRoutesQuery{}, err
	}

	query.TokenIn = wrappedTokenIn
	query.TokenOut = wrappedTokenOut

	return query, nil
}

func (u *useCase) getAggregateParams(ctx context.Context, query dto.GetCustomRoutesQuery) (*types.AggregateParams,
	error) {
	tokenByAddress, err := u.getTokenByAddress(ctx, query.TokenIn, query.TokenOut, u.config.GasTokenAddress)
	if err != nil {
		return nil, err
	}

	tokenIn, ok := tokenByAddress[query.TokenIn]
	if !ok {
		return nil, errors.WithMessagef(getroute.ErrTokenNotFound, "tokenIn: [%s]", query.TokenIn)
	}

	tokenOut, ok := tokenByAddress[query.TokenOut]
	if !ok {
		return nil, errors.WithMessagef(getroute.ErrTokenNotFound, "tokenOut: [%s]", query.TokenOut)
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
			return nil, err
		}
	}

	return &types.AggregateParams{
		TokenIn:          tokenIn,
		TokenOut:         tokenOut,
		GasToken:         tokenByAddress[u.config.GasTokenAddress],
		TokenInPriceUSD:  tokenInPriceUSD,
		TokenOutPriceUSD: tokenOutPriceUSD,
		GasTokenPriceUSD: gasTokenPriceUSD,
		AmountIn:         query.AmountIn,
		Sources: u.getSources(query.ClientId, query.IncludedSources, query.ExcludedSources,
			query.OnlyScalableSources),
		SaveGas:       query.SaveGas,
		GasInclude:    query.GasInclude,
		GasPrice:      gasPrice,
		L1FeeOverhead: l1FeeOverhead,
		L1FeePerPool:  l1FeePerPool,
		ExtraFee:      query.ExtraFee,
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
