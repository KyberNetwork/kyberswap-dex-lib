package getcustomroute

import (
	"context"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
)

type useCase struct {
	aggregator IAggregator

	tokenRepository ITokenRepository
	priceRepository IPriceRepository
	gasRepository   IGasRepository

	onchainpriceRepository IOnchainPriceRepository

	config Config
	mu     sync.Mutex
}

func NewCustomRoutesUseCase(
	poolFactory IPoolFactory,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	onchainpriceRepository IOnchainPriceRepository,
	gasRepository IGasRepository,
	poolRepository IPoolRepository,
	finderEngine finderEngine.IPathFinderEngine,
	config Config,
) *useCase {
	aggregator := NewCustomAggregator(
		poolFactory,
		tokenRepository,
		priceRepository,
		onchainpriceRepository,
		poolRepository,
		config.Aggregator,
		finderEngine,
	)

	return &useCase{
		aggregator:      aggregator,
		tokenRepository: tokenRepository,
		priceRepository: priceRepository,
		gasRepository:   gasRepository,
		config:          config,

		onchainpriceRepository: onchainpriceRepository,
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

func (u *useCase) getAggregateParams(ctx context.Context, query dto.GetCustomRoutesQuery) (*types.AggregateParams, error) {
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

	tokenInPriceUSD, tokenOutPriceUSD, gasTokenPriceUSD, err := u.getTokensPriceUSD(ctx, query.TokenIn, query.TokenOut, u.config.GasTokenAddress)
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
		TokenInPriceUSD:  tokenInPriceUSD,
		TokenOutPriceUSD: tokenOutPriceUSD,
		GasTokenPriceUSD: gasTokenPriceUSD,
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

func (u *useCase) getTokensPriceUSD(ctx context.Context, tokenIn, tokenOut, gasToken string) (float64, float64, float64, error) {
	if u.onchainpriceRepository != nil {
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

	// fallback to legacy price-service
	priceUSDByAddress, err := u.getPriceUSDByAddress(ctx, tokenIn, tokenOut, gasToken)
	if err != nil {
		return 0, 0, 0, err
	}
	return priceUSDByAddress[tokenIn],
		priceUSDByAddress[tokenOut],
		priceUSDByAddress[gasToken], nil
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
