package getroute

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// aggregator finds best route within amm liquidity sources
type aggregator struct {
	config AggregatorConfig

	poolRankRepository     IPoolRankRepository
	tokenRepository        ITokenRepository
	onchainPriceRepository IOnchainPriceRepository
	poolManager            IPoolManager
	finderEngine           finderEngine.IPathFinderEngine
}

func NewAggregator(
	config AggregatorConfig,
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	onchainPriceRepository IOnchainPriceRepository,
	poolManager IPoolManager,
	finderEngine finderEngine.IPathFinderEngine,
) *aggregator {
	return &aggregator{
		config:                 config,
		poolRankRepository:     poolRankRepository,
		tokenRepository:        tokenRepository,
		onchainPriceRepository: onchainPriceRepository,
		poolManager:            poolManager,
		finderEngine:           finderEngine,
	}
}

func (a *aggregator) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummaries,
	error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] aggregator.Aggregate")
	defer span.End()

	// Step 1: get pool set
	state, err := a.getStateByAddress(ctx, params)
	if err != nil {
		return nil, err
	}

	// Step 2: collect tokens and price data
	tokenAddresses := lo.Keys(a.config.WhitelistedTokenSet)
	tokenAddresses = append(
		tokenAddresses,
		params.TokenIn.Address,
		params.TokenOut.Address,
		params.GasToken.Address,
	)

	tokenByAddress, err := a.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	// only get price from onchain-price-service if enabled
	priceByAddress, err := a.onchainPriceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	// Step 3: finds best route
	return a.findBestRoute(ctx, params, tokenByAddress, priceByAddress, state)
}

func (a *aggregator) ApplyConfig(config Config) {
	a.config = config.Aggregator
}

// findBestRoute find the best route and summarize it
func (a *aggregator) findBestRoute(
	ctx context.Context,
	params *types.AggregateParams,
	tokenByAddress map[string]*entity.SimplifiedToken,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
) (*valueobject.RouteSummaries, error) {
	findRouteParams := ConvertToPathfinderParams(
		a.config.WhitelistedTokenSet,
		params,
		tokenByAddress,
		priceByAddress,
		state,
		a.config.FeatureFlags,
	)

	routes, err := a.finderEngine.Find(ctx, findRouteParams)

	if err != nil {
		if errors.Is(err, finderEngine.ErrInvalidSwap) {
			return nil, errors.WithMessagef(ErrInvalidSwap, "find route failed: [%v]", err)
		} else if errors.Is(err, finderEngine.ErrRouteNotFound) {
			return nil, errors.WithMessagef(ErrRouteNotFound, "find route failed: [%v]", err)
		} else {
			return nil, err
		}
	}

	// We don't expect this logic happens but safe check and log here
	if routes.GetBestRoute() == nil {
		return nil, errors.WithMessagef(ErrRouteNotFound, "best route is nil")
	}

	return ConvertToRouteSummaries(params, routes), nil
}

func (a *aggregator) getStateByAddress(ctx context.Context, params *types.AggregateParams) (*types.FindRouteState,
	error) {
	if len(params.Sources) == 0 {
		log.Ctx(ctx).Err(ErrPoolSetFiltered).Msg("sources list is empty")
		return nil, ErrPoolSetFiltered
	}
	var bestPoolIDs []string
	var err error

	opt := a.config.GetBestPoolsOptions
	opt.OnlyDirectPools = params.OnlyDirectPools

	if len(params.PoolIds) > 0 {
		bestPoolIDs = params.PoolIds
	} else {
		bestPoolIDs, err = a.poolRankRepository.FindBestPoolIDs(
			ctx,
			params.TokenIn.Address,
			params.TokenOut.Address,
			params.AmountInUsd,
			opt,
			params.Index,
			params.ForcePoolsForToken,
		)
		if err != nil {
			return nil, err
		}
	}

	if len(bestPoolIDs) == 0 {
		return nil, ErrPoolSetEmpty
	}

	filteredPoolIDs := make([]string, 0, len(bestPoolIDs))
	for _, bestPoolID := range bestPoolIDs {
		if params.ExcludedPools != nil && params.ExcludedPools.Contains(bestPoolID) {
			continue
		}
		filteredPoolIDs = append(filteredPoolIDs, bestPoolID)
	}

	if len(filteredPoolIDs) == 0 {
		log.Ctx(ctx).Err(ErrPoolSetFiltered).Msgf("empty filtered pool IDs. bestPoolIDs %v, excludedPools: %v",
			bestPoolIDs, params.ExcludedPools)
		return nil, ErrPoolSetFiltered
	}

	state, err := a.poolManager.GetStateByPoolAddresses(ctx, filteredPoolIDs, params.Sources,
		types.PoolManagerExtraData{
			KyberLimitOrderAllowedSenders: params.KyberLimitOrderAllowedSenders,
		})
	if err != nil {
		return nil, err
	}
	forcePoolsForToken(state, params.TokenIn.Address, params.TokenOut.Address, params.ForcePoolsForToken)

	return state, nil
}

// getTokenByAddress receives a list of address and returns a map of address to entity.SimplifiedToken
func (a *aggregator) getTokenByAddress(ctx context.Context,
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

func forcePoolsForToken(state *types.FindRouteState, tokenIn, tokenOut string, forcePoolsForToken map[string][]string) {
	if forcePoolsForTokenIn := forcePoolsForToken[tokenIn]; len(forcePoolsForTokenIn) > 0 {
		for _, pool := range state.Pools {
			if pool.GetTokenIndex(tokenIn) >= 0 {
				poolAddr := pool.GetAddress()
				if !lo.Contains(forcePoolsForTokenIn, poolAddr) {
					delete(state.Pools, poolAddr)
				}
			}
		}
	}
	if forcePoolsForTokenOut := forcePoolsForToken[tokenOut]; len(forcePoolsForTokenOut) > 0 {
		for _, pool := range state.Pools {
			if pool.GetTokenIndex(tokenOut) >= 0 {
				poolAddr := pool.GetAddress()
				if !lo.Contains(forcePoolsForTokenOut, poolAddr) {
					delete(state.Pools, poolAddr)
				}
			}
		}
	}
}
