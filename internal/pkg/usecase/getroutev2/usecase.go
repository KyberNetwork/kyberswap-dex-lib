package getroutev2

import (
	"context"
	"math/big"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
)

type useCase struct {
	aggregator IAggregator

	tokenRepository ITokenRepository
	priceRepository IPriceRepository
	gasRepository   IGasRepository

	config Config
}

func NewUseCase(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	routeCacheRepository IRouteCacheRepository,
	gasRepository IGasRepository,
	poolRepository IPoolRepository,
	config Config,
) *useCase {
	poolFactory := NewPoolFactory(config.PoolFactory)
	poolManager := NewPoolManager(poolRepository, poolFactory, config.PoolManager)
	ammAggregator := NewAMMAggregator(
		poolRankRepository,
		tokenRepository,
		priceRepository,
		poolManager,
		config.AmmAggregator,
	)
	aggregatorWithCache := NewCache(ammAggregator, routeCacheRepository, poolManager, config.Cache)
	aggregatorWitchChargeExtraFee := NewChargeExtraFee(aggregatorWithCache)

	return &useCase{
		aggregator:      aggregatorWitchChargeExtraFee,
		tokenRepository: tokenRepository,
		priceRepository: priceRepository,
		gasRepository:   gasRepository,
		config:          config,
	}
}

func (u *useCase) Handle(ctx context.Context, query dto.GetRoutesQuery) (*dto.GetRoutesResult, error) {
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

	routeSummary, err := u.aggregator.Aggregate(ctx, params)
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

func (u *useCase) getAggregateParams(ctx context.Context, query dto.GetRoutesQuery) (*types.AggregateParams, error) {
	tokenByAddress, err := u.getTokenByAddress(ctx, query.TokenIn, query.TokenOut, u.config.GasTokenAddress)
	if err != nil {
		return nil, err
	}

	tokenIn, ok := tokenByAddress[query.TokenIn]
	if !ok {
		return nil, errors.Wrapf(ErrTokenNotFound, "tokenIn: [%s]", query.TokenIn)
	}

	tokenOut, ok := tokenByAddress[query.TokenOut]
	if !ok {
		return nil, errors.Wrapf(ErrTokenNotFound, "tokenOut: [%s]", query.TokenIn)
	}

	priceUSDByAddress, err := u.getPriceUSDByAddress(ctx, query.TokenIn, query.TokenOut, u.config.GasTokenAddress)
	if err != nil {
		return nil, err
	}

	gasPrice, err := u.getGasPrice(ctx, query.GasPrice)
	if err != nil {
		return nil, err
	}

	return &types.AggregateParams{
		TokenIn:          tokenIn,
		TokenOut:         tokenOut,
		GasToken:         tokenByAddress[u.config.GasTokenAddress],
		TokenInPriceUSD:  priceUSDByAddress[query.TokenIn],
		TokenOutPriceUSD: priceUSDByAddress[query.TokenOut],
		GasTokenPriceUSD: priceUSDByAddress[u.config.GasTokenAddress],
		AmountIn:         query.AmountIn,
		Sources:          u.getSources(query.IncludedSources, query.ExcludedSources),
		SaveGas:          query.SaveGas,
		GasInclude:       query.GasInclude,
		GasPrice:         gasPrice,
		ExtraFee:         query.ExtraFee,
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

func (u *useCase) getPriceUSDByAddress(ctx context.Context, addresses ...string) (map[string]float64, error) {
	prices, err := u.priceRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	priceUSDByAddress := make(map[string]float64, len(prices))
	for _, price := range prices {
		priceUSD, _ := price.GetPreferredPrice()

		priceUSDByAddress[price.Address] = priceUSD
	}

	return priceUSDByAddress, nil
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

func (u *useCase) getSources(includedSources []string, excludedSources []string) []string {
	sources := make([]string, 0, len(u.config.AvailableSources))
	includedSourcesLen := len(includedSources)
	excludedSourcesLen := len(excludedSources)

	for _, source := range u.config.AvailableSources {
		if excludedSourcesLen > 0 && utils.StringContains(excludedSources, source) {
			continue
		}

		if includedSourcesLen > 0 && !utils.StringContains(includedSources, source) {
			continue
		}

		sources = append(sources, source)
	}

	return sources
}
