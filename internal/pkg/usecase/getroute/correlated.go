package getroute

import (
	"context"
	"slices"
	"sync"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type correlatedPairs struct {
	aggregator IAggregator

	oneAdditionHopFinderEngine  finderEngine.IPathFinderEngine
	twoAdditionHopsFinderEngine finderEngine.IPathFinderEngine

	poolRankRepository     IPoolRankRepository
	tokenRepository        ITokenRepository
	priceRepository        IPriceRepository
	onchainPriceRepository IOnchainPriceRepository
	poolManager            IPoolManager

	// map[token0] -> map[token1] -> poolAddress
	correlatedPairs map[string]map[string]string

	config Config
	mu     sync.RWMutex
}

func NewCorrelatedPairs(
	aggregator IAggregator,
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	onchainPriceRepository IOnchainPriceRepository,
	poolManager IPoolManager,
	config Config,
) *correlatedPairs {
	oneAdditionHopFinderEngine, twoAdditionHopsFinderEngine := initAdditionHopFinderEngines(config, poolManager.GetAEVMClient())

	return &correlatedPairs{
		aggregator: aggregator,

		oneAdditionHopFinderEngine:  oneAdditionHopFinderEngine,
		twoAdditionHopsFinderEngine: twoAdditionHopsFinderEngine,

		poolRankRepository:     poolRankRepository,
		tokenRepository:        tokenRepository,
		priceRepository:        priceRepository,
		onchainPriceRepository: onchainPriceRepository,
		poolManager:            poolManager,

		correlatedPairs: convertCorrelatedPairsMap(config.CorrelatedPairs),
		config:          config,
	}
}

func (c *correlatedPairs) Aggregate(
	ctx context.Context,
	params *types.AggregateParams,
) (*valueobject.RouteSummary, error) {
	baseRoute, aggregateErr := c.aggregator.Aggregate(ctx, params)
	if baseRoute != nil && aggregateErr == nil {
		return baseRoute, nil
	}

	// BaseAggregator can't find route. We will try to find route with correlated pairs

	poolInAddress, tokenMidIn := c.findFirstCorrelatedPool(params.TokenIn.Address)
	poolOutAddress, tokenMidOut := c.findFirstCorrelatedPool(params.TokenOut.Address)

	additionPoolAddresses := slices.DeleteFunc([]string{poolInAddress, poolOutAddress}, func(s string) bool { return s == "" })
	additionHops := len(additionPoolAddresses)
	if additionHops == 0 {
		// Can't find any correlated pairs. Return the base result.
		return nil, aggregateErr
	}

	// Initialize find route data
	var stateRoot aevmcommon.Hash
	var err error
	if aevmClient := c.poolManager.GetAEVMClient(); aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot(ctx)
		if err != nil {
			return nil, err
		}
	}

	state, err := c.getStateByAddress(ctx, params, tokenMidIn, tokenMidOut, additionPoolAddresses, common.Hash(stateRoot))
	if err != nil {
		return nil, err
	}

	tokenAddresses := lo.Keys(c.config.Aggregator.WhitelistedTokenSet)
	tokenAddresses = append(
		tokenAddresses,
		params.TokenIn.Address,
		params.TokenOut.Address,
		params.GasToken.Address,
		tokenMidIn,
		tokenMidOut,
	)

	tokens, err := c.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	var usdPrices map[string]float64
	var onchainPrices map[string]*routerEntity.OnchainPrice
	if c.onchainPriceRepository != nil {
		onchainPrices, err = c.onchainPriceRepository.FindByAddresses(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}
	} else {
		usdPrices, err = c.getPriceUSDByAddress(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}
	}

	whitelistTokens := map[string]bool{}
	whitelistTokens[tokenMidIn] = true
	whitelistTokens[tokenMidOut] = true
	for token := range c.config.Aggregator.WhitelistedTokenSet {
		whitelistTokens[token] = true
	}

	// Find route from tokenIn -> tokenOut
	findRouteParams := ConvertToPathfinderParams(
		whitelistTokens,
		params,
		tokens,
		usdPrices,
		onchainPrices,
		state,
	)

	var route *finderEntity.Route

	if additionHops == 1 {
		route, err = c.oneAdditionHopFinderEngine.Find(ctx, findRouteParams)
	} else if additionHops == 2 {
		route, err = c.twoAdditionHopsFinderEngine.Find(ctx, findRouteParams)
	}

	if err != nil {
		return nil, errors.WithMessagef(ErrRouteNotFound, "find route failed: [%v]", err)
	}

	return ConvertToRouteSummary(params, route), nil
}

func (c *correlatedPairs) ApplyConfig(config Config) {
	c.mu.Lock()
	c.config = config
	c.mu.Unlock()

	c.correlatedPairs = convertCorrelatedPairsMap(config.CorrelatedPairs)
	oneAdditionHopFinderEngine, twoAdditionHopsFinderEngine := initAdditionHopFinderEngines(config, c.poolManager.GetAEVMClient())
	c.oneAdditionHopFinderEngine = oneAdditionHopFinderEngine
	c.twoAdditionHopsFinderEngine = twoAdditionHopsFinderEngine
}

func (c *correlatedPairs) findFirstCorrelatedPool(token string) (string, string) {
	// For now, we always config 1 correlatedTokenIn -> 1 correlatedTokenOut through 1 pool.
	// So return the first possible pool.
	for tokenOut, poolAddress := range c.correlatedPairs[token] {
		return poolAddress, tokenOut
	}

	// No correlated pairs found. Fallback to same token.
	return "", token
}

func (c *correlatedPairs) getStateByAddress(
	ctx context.Context,
	params *types.AggregateParams,
	tokenMidIn string,
	tokenMidOut string,
	additionPoolAddresses []string,
	stateRoot common.Hash,
) (*types.FindRouteState, error) {
	if len(params.Sources) == 0 {
		return nil, ErrPoolSetFiltered
	}

	bestPoolIDs, err := c.poolRankRepository.FindBestPoolIDs(
		ctx,
		tokenMidIn,
		tokenMidOut,
		c.config.Aggregator.GetBestPoolsOptions,
	)
	if err != nil {
		return nil, err
	}

	if len(bestPoolIDs) == 0 {
		return nil, ErrPoolSetEmpty
	}

	bestPoolIDs = append(bestPoolIDs, additionPoolAddresses...)

	filteredPoolIDs := make([]string, 0, len(bestPoolIDs))
	for _, bestPoolID := range bestPoolIDs {
		if params.ExcludedPools != nil && params.ExcludedPools.Contains(bestPoolID) {
			continue
		}
		filteredPoolIDs = append(filteredPoolIDs, bestPoolID)
	}

	if len(filteredPoolIDs) == 0 {
		return nil, ErrPoolSetFiltered
	}

	return c.poolManager.GetStateByPoolAddresses(
		ctx,
		filteredPoolIDs,
		params.Sources,
		stateRoot,
	)
}

func (a *correlatedPairs) getTokenByAddress(ctx context.Context, tokenAddresses []string) (map[string]*entity.Token, error) {
	tokens, err := a.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]*entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = token
	}

	return tokenByAddress, nil
}

func (a *correlatedPairs) getPriceUSDByAddress(ctx context.Context, tokenAddresses []string) (map[string]float64, error) {
	prices, err := a.priceRepository.FindByAddresses(ctx, tokenAddresses)
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
