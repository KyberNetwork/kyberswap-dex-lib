package getcustomroute

import (
	"context"
	"fmt"
	"math/big"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
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
	poolEntities, err := a.poolRepository.FindByAddresses(ctx, poolIds)
	if err != nil {
		return nil, err
	}

	if len(poolEntities) == 0 {
		return nil, getroute.ErrPoolSetEmpty
	}

	var stateRoot aevmcommon.Hash
	if aevmClient := a.poolManager.GetAEVMClient(); aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot(ctx)
		if err != nil {
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}
	poolByAddress := make(map[string]poolpkg.IPoolSimulator, len(poolIds))
	poolInterfaces := a.poolFactory.NewPools(ctx, poolEntities, common.Hash(stateRoot))
	for i := range poolInterfaces {
		poolByAddress[poolInterfaces[i].GetAddress()] = poolInterfaces[i]
	}

	if len(poolByAddress) == 0 {
		return nil, getroute.ErrPoolSetFiltered
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

	var limits = make(map[string]map[string]*big.Int)
	for _, poolType := range constant.DexUseSwapLimit {
		limits[poolType] = make(map[string]*big.Int)
	}
	for _, pool := range poolInterfaces {
		dexLimit, avail := limits[pool.GetType()]
		if !avail {
			continue
		}
		limitMap := pool.CalculateLimit()
		for k, v := range limitMap {
			if old, exist := dexLimit[k]; !exist || old.Cmp(v) < 0 {
				dexLimit[k] = v
			}
		}
	}

	// Step 3: finds best route
	return a.findBestRoute(ctx, params, tokenByAddress, priceByAddress, &types.FindRouteState{
		Pools:     poolByAddress,
		SwapLimit: a.poolFactory.NewSwapLimit(limits, types.PoolManagerExtraData{}),
	})
}

func (a *aggregator) ApplyConfig(config getroute.Config) {}

// findBestRoute find the best route and summarize it
func (a *aggregator) findBestRoute(
	ctx context.Context,
	params *types.AggregateParams,
	tokenByAddress map[string]*entity.Token,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
) (*valueobject.RouteSummaries, error) {
	findRouteParams := getroute.ConvertToPathfinderParams(
		a.config.WhitelistedTokenSet,
		params,
		tokenByAddress,
		priceByAddress,
		state,
	)
	findRouteParams.SkipMergeSwap = !a.config.FeatureFlags.IsMergeDuplicateSwapEnabled || params.IsScaleHelperClient

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

// getTokenByAddress receives a list of address and returns a map of address to entity.Token
func (a *aggregator) getTokenByAddress(ctx context.Context, tokenAddresses []string) (map[string]*entity.Token, error) {
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
