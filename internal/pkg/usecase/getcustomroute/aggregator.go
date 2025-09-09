package getcustomroute

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type aggregator struct {
	poolFactory            getroute.IPoolFactory
	tokenRepository        getroute.ITokenRepository
	onchainpriceRepository getroute.IOnchainPriceRepository
	poolManager            getroute.IPoolManager
	poolRepository         getroute.IPoolRepository

	finderEngine finderEngine.IPathFinderEngine

	config getroute.AggregatorConfig
}

func NewCustomAggregator(
	poolFactory getroute.IPoolFactory,
	tokenRepository getroute.ITokenRepository,
	onchainpriceRepository getroute.IOnchainPriceRepository,
	poolManager getroute.IPoolManager,
	poolRepository getroute.IPoolRepository,
	config getroute.AggregatorConfig,
	finderEngine finderEngine.IPathFinderEngine,
) *aggregator {
	return &aggregator{
		poolFactory:            poolFactory,
		tokenRepository:        tokenRepository,
		onchainpriceRepository: onchainpriceRepository,
		poolManager:            poolManager,
		poolRepository:         poolRepository,
		finderEngine:           finderEngine,
		config:                 config,
	}
}

func (a *aggregator) Aggregate(ctx context.Context, params *types.AggregateParams,
	poolIds []string) (*valueobject.RouteSummaries, error) {
	// Step 1: get pool set
	state, err := a.poolManager.GetStateByPoolAddresses(ctx, poolIds, params.Sources,
		types.PoolManagerExtraData{
			KyberLimitOrderAllowedSenders: params.KyberLimitOrderAllowedSenders,
		})
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
	priceByAddress, err := a.onchainpriceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	// Step 3: finds best route
	return a.findBestRoute(ctx, params, tokenByAddress, priceByAddress, &types.FindRouteState{
		Pools:     state.Pools,
		SwapLimit: state.SwapLimit,
	})
}

func (a *aggregator) ApplyConfig(config getroute.Config) {
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
	findRouteParams := getroute.ConvertToPathfinderParams(
		a.config.WhitelistedTokenSet,
		params,
		tokenByAddress,
		priceByAddress,
		state,
		a.config.FeatureFlags,
		&a.config,
	)

	routes, err := a.finderEngine.Find(ctx, findRouteParams)

	if err != nil {
		if errors.Is(err, finderEngine.ErrInvalidSwap) {
			return nil, errors.WithMessagef(getroute.ErrInvalidSwap, "find route failed: [%v]", err)
		} else if errors.Is(err, finderEngine.ErrRouteNotFound) {
			return nil, errors.WithMessagef(getroute.ErrRouteNotFound, "find route failed: [%v]", err)
		} else {
			return nil, err
		}
	}

	// We don't expect this logic happens but safe check and log here
	if routes.GetBestRoute() == nil {
		return nil, errors.WithMessagef(getroute.ErrRouteNotFound, "bet route is nil")
	}

	return getroute.ConvertToRouteSummaries(params, routes), nil
}

// getTokenByAddress receives a list of address and returns a map of address to entity.SimplifiedToken
func (a *aggregator) getTokenByAddress(ctx context.Context, tokenAddresses []string) (map[string]*entity.SimplifiedToken, error) {
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
