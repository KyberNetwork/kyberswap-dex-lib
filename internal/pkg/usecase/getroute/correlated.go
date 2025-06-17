package getroute

import (
	"context"
	"slices"
	"sync"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type correlatedPairs struct {
	aggregator IAggregator

	oneAdditionHopFinderEngine  finderEngine.IPathFinderEngine
	twoAdditionHopsFinderEngine finderEngine.IPathFinderEngine

	poolRankRepository     IPoolRankRepository
	tokenRepository        ITokenRepository
	onchainPriceRepository IOnchainPriceRepository
	poolManager            IPoolManager
	aevmClient             aevmclient.Client

	// map[token0] -> map[token1] -> poolAddress
	correlatedPairs map[string]map[string]string

	config Config
	mu     sync.RWMutex
}

func NewCorrelatedPairs(
	aggregator IAggregator,
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	onchainPriceRepository IOnchainPriceRepository,
	poolManager IPoolManager,
	aevmClient aevmclient.Client,
	config Config,
) *correlatedPairs {
	oneAdditionHopFinderEngine, twoAdditionHopsFinderEngine := initAdditionHopFinderEngines(config, aevmClient)

	return &correlatedPairs{
		aggregator: aggregator,

		oneAdditionHopFinderEngine:  oneAdditionHopFinderEngine,
		twoAdditionHopsFinderEngine: twoAdditionHopsFinderEngine,

		poolRankRepository:     poolRankRepository,
		tokenRepository:        tokenRepository,
		onchainPriceRepository: onchainPriceRepository,
		poolManager:            poolManager,
		aevmClient:             aevmClient,

		correlatedPairs: convertCorrelatedPairsMap(config.CorrelatedPairs),
		config:          config,
	}
}

func (c *correlatedPairs) Aggregate(
	ctx context.Context,
	params *types.AggregateParams,
) (*valueobject.RouteSummaries, error) {
	baseRoute, aggregateErr := c.aggregator.Aggregate(ctx, params)
	if aggregateErr == nil && baseRoute.GetBestRouteSummary() != nil {
		return baseRoute, nil
	}

	// BaseAggregator can't find route. We will try to find route with correlated pairs

	poolInAddress, tokenMidIn := c.findFirstCorrelatedPool(params.TokenIn.Address)
	poolOutAddress, tokenMidOut := c.findFirstCorrelatedPool(params.TokenOut.Address)

	additionPoolAddresses := slices.DeleteFunc([]string{poolInAddress, poolOutAddress},
		func(s string) bool { return s == "" })
	additionHops := len(additionPoolAddresses)
	if additionHops == 0 {
		// Can't find any correlated pairs. Return the base result.
		return nil, aggregateErr
	}

	// Initialize find route data
	state, err := c.getStateByAddress(ctx, params, tokenMidIn, tokenMidOut, additionPoolAddresses)
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

	onchainPrices, err := c.onchainPriceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
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
		onchainPrices,
		state,
		c.config.FeatureFlags,
	)

	var routes finderEntity.BestRoutes

	if additionHops == 1 {
		routes, err = c.oneAdditionHopFinderEngine.Find(ctx, findRouteParams)
	} else if additionHops == 2 {
		routes, err = c.twoAdditionHopsFinderEngine.Find(ctx, findRouteParams)
	}

	if err != nil || routes.GetBestRoute() == nil {
		return nil, errors.WithMessagef(ErrRouteNotFound, "find route failed: [%v]", err)
	}

	return ConvertToRouteSummaries(params, routes), nil
}

func (c *correlatedPairs) ApplyConfig(config Config) {
	c.mu.Lock()
	c.config = config
	c.mu.Unlock()

	c.correlatedPairs = convertCorrelatedPairsMap(config.CorrelatedPairs)
	oneAdditionHopFinderEngine, twoAdditionHopsFinderEngine := initAdditionHopFinderEngines(config, c.aevmClient)
	c.oneAdditionHopFinderEngine = oneAdditionHopFinderEngine
	c.twoAdditionHopsFinderEngine = twoAdditionHopsFinderEngine

	c.aggregator.ApplyConfig(config)
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

func (c *correlatedPairs) getStateByAddress(ctx context.Context, params *types.AggregateParams, tokenMidIn string,
	tokenMidOut string, additionPoolAddresses []string) (*types.FindRouteState, error) {
	if len(params.Sources) == 0 {
		logger.Errorf(ctx, "sources list is empty, returning error: %v", ErrPoolSetFiltered)
		return nil, ErrPoolSetFiltered
	}

	var bestPoolIDs []string
	var err error
	bestPoolIDs, err = c.poolRankRepository.FindBestPoolIDs(
		ctx,
		tokenMidIn,
		tokenMidOut,
		params.AmountInUsd,
		c.config.Aggregator.GetBestPoolsOptions,
		params.Index,
		params.ForcePoolsForToken,
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
		logger.Errorf(ctx,
			"empty filtered pool IDs after excluding pools, returning error: %v, bestPoolIDs: %v, index: %v",
			ErrPoolSetFiltered, bestPoolIDs, params.Index)
		return nil, ErrPoolSetFiltered
	}

	state, err := c.poolManager.GetStateByPoolAddresses(ctx, filteredPoolIDs, params.Sources,
		types.PoolManagerExtraData{
			KyberLimitOrderAllowedSenders: params.KyberLimitOrderAllowedSenders,
		})
	if err != nil {
		return nil, err
	}
	forcePoolsForToken(state, params.TokenIn.Address, params.TokenOut.Address, params.ForcePoolsForToken)

	return state, nil
}

func (a *correlatedPairs) getTokenByAddress(ctx context.Context,
	tokenAddresses []string) (map[string]*entity.SimplifiedToken, error) {
	tokens, err := a.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]*entity.SimplifiedToken, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = token
	}

	return tokenByAddress, nil
}
