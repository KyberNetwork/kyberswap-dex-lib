package getroute

import (
	"context"
	"fmt"
	"maps"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type bundledAggregator struct {
	*aggregator

	poolFactory IPoolFactory
}

func NewBundledAggregator(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	onchainpriceRepository IOnchainPriceRepository,
	poolManager IPoolManager,
	poolFactory IPoolFactory,
	config AggregatorConfig,
	finderEngine finderEngine.IPathFinderEngine,
) *bundledAggregator {
	ag := &aggregator{
		poolRankRepository:     poolRankRepository,
		tokenRepository:        tokenRepository,
		onchainpriceRepository: onchainpriceRepository,
		poolManager:            poolManager,
		finderEngine:           finderEngine,
		config:                 config,
	}
	return &bundledAggregator{ag, poolFactory}
}

func (a *bundledAggregator) ApplyConfig(config Config) {
	a.aggregator.ApplyConfig(config)
}

func (a *bundledAggregator) Aggregate(ctx context.Context, params *types.AggregateBundledParams) ([]*valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] aggregator.AggregateBundled")
	defer span.End()

	if len(params.Pairs) == 0 {
		return nil, ErrNoPair
	}

	// Step 1: get pool set
	var (
		stateRoot aevmcommon.Hash
		err       error
	)
	if aevmClient := a.poolManager.GetAEVMClient(); aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot(ctx)
		if err != nil {
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}

	// Step 2: collect tokens and price data
	tokenAddresses := lo.Keys(a.config.WhitelistedTokenSet)
	for _, pair := range params.Pairs {
		tokenAddresses = append(tokenAddresses, pair.TokenIn, pair.TokenOut)
	}
	tokenAddresses = append(tokenAddresses, params.GasToken)

	tokenByAddress, err := a.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	// only get price from onchain-price-service if enabled
	priceByAddress, err := a.onchainpriceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}
	// Calculate amountInUsd for every pair to find best pools
	for _, pair := range params.Pairs {
		tokenIn, ok := tokenByAddress[pair.TokenIn]
		if !ok {
			return nil, errors.WithMessagef(ErrInvalidToken, "invalid tokenIn: %v", pair.TokenIn)
		}
		tokenInPrice := GetPrice(pair.TokenIn, priceByAddress, false)
		pair.AmountInUsd = utils.CalcTokenAmountUsd(pair.AmountIn, tokenIn.Decimals, tokenInPrice)
		if pair.AmountInUsd > MaxAmountInUSD {
			return nil, ErrAmountInIsGreaterThanMaxAllowed
		}
	}

	state, err := a.getStateByBundledAddress(ctx, params, common.Hash(stateRoot))
	if err != nil {
		return nil, err
	}

	// override pool if requested
	if len(params.OverridePools) > 0 {
		// create pool simulators from override pools
		// if caller want to override a curve meta pool, they need to supply override state for its base pool as well
		poolSims := a.poolFactory.NewPoolByAddress(ctx, params.OverridePools, common.Hash(stateRoot))

		for _, pool := range params.OverridePools {
			if len(pool.Address) == 0 {
				continue
			}
			poolSim := poolSims[pool.Address]
			if poolSim == nil {
				logger.Errorf(ctx, "could not get pool simulator for pool %v", pool.Address)
				delete(state.Pools, pool.Address)
				continue
			}
			logger.Debugf(ctx, "overriding pool %v: %v | %v", pool.Address, state.Pools[pool.Address], poolSim)
			state.Pools[pool.Address] = poolSim
		}

		// if caller want to override a curve base pool, then we need to find its meta pool in `state` and update basepool there
		newMetaPools := a.poolFactory.CloneMetaPoolsWithBasePools(ctx, state.Pools, poolSims)
		for _, newMetaPool := range newMetaPools {
			addr := newMetaPool.GetAddress()
			if _, ok := state.Pools[addr]; ok {
				logger.Debugf(ctx, "overriding meta pool %v | %v", state.Pools[addr], newMetaPool)
				state.Pools[addr] = newMetaPool
			}
		}
	}

	// Step 3: finds best route
	return a.findBestBundledRoute(ctx, params, tokenByAddress, priceByAddress, state)
}

func (a *bundledAggregator) getStateByBundledAddress(
	ctx context.Context,
	params *types.AggregateBundledParams,
	stateRoot common.Hash,
) (*types.FindRouteState, error) {
	if len(params.Sources) == 0 {
		return nil, ErrPoolSetFiltered
	}

	var bestPoolIDs []string
	for _, pair := range params.Pairs {
		pairPoolIDs, err := a.poolRankRepository.FindBestPoolIDs(
			ctx, pair.TokenIn, pair.TokenOut, pair.AmountInUsd, a.config.GetBestPoolsOptions, params.Index)

		if err != nil {
			return nil, err
		}
		bestPoolIDs = append(bestPoolIDs, pairPoolIDs...)
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
		logger.Errorf(ctx, "empty filtered pool IDs. bestPoolIDs %v, excludedPools: %v",
			bestPoolIDs, params.ExcludedPools.String())
		return nil, ErrPoolSetFiltered
	}

	return a.poolManager.GetStateByPoolAddresses(
		ctx,
		filteredPoolIDs,
		params.Sources,
		stateRoot,
		types.PoolManagerExtraData{
			KyberLimitOrderAllowedSenders: params.KyberLimitOrderAllowedSenders,
		},
	)
}

func (a *bundledAggregator) findBestBundledRoute(
	ctx context.Context,
	params *types.AggregateBundledParams,
	tokenByAddress map[string]*entity.SimplifiedToken,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
) ([]*valueobject.RouteSummary, error) {
	allRoutes := make([]*valueobject.RouteSummary, 0, len(params.Pairs))

	var lastSwapState *types.StateAfterSwap

	gasToken, ok := tokenByAddress[params.GasToken]
	if !ok {
		return nil, errors.WithMessagef(ErrInvalidToken, "invalid gasToken: %v", params.GasToken)
	}
	gasTokenPrice := GetPrice(params.GasToken, priceByAddress, true)

	whitelistedTokenSet := a.config.WhitelistedTokenSet
	if len(params.ExtraWhitelistedTokens) > 0 {
		whitelistedTokenSet = maps.Clone(a.config.WhitelistedTokenSet)
		for _, token := range params.ExtraWhitelistedTokens {
			whitelistedTokenSet[token] = true
		}
	}

	for _, pair := range params.Pairs {
		tokenIn, ok := tokenByAddress[pair.TokenIn]
		if !ok {
			return nil, errors.WithMessagef(ErrInvalidToken, "invalid tokenIn: %v", pair.TokenIn)
		}
		tokenOut, ok := tokenByAddress[pair.TokenOut]
		if !ok {
			return nil, errors.WithMessagef(ErrInvalidToken, "invalid tokenOut: %v", pair.TokenOut)
		}
		tokenInPrice := GetPrice(pair.TokenIn, priceByAddress, false)  // use sell price for tokenIn
		tokenOutPrice := GetPrice(pair.TokenOut, priceByAddress, true) // use buy price for token out and gas

		pairParams := types.AggregateParams{
			TokenIn:                       *tokenIn,
			TokenOut:                      *tokenOut,
			GasToken:                      *gasToken,
			TokenInPriceUSD:               tokenInPrice,
			TokenOutPriceUSD:              tokenOutPrice,
			GasTokenPriceUSD:              gasTokenPrice,
			AmountIn:                      pair.AmountIn,
			AmountInUsd:                   pair.AmountInUsd,
			Sources:                       params.Sources,
			GasInclude:                    params.GasInclude,
			GasPrice:                      params.GasPrice,
			L1FeeOverhead:                 params.L1FeeOverhead,
			L1FeePerPool:                  params.L1FeePerPool,
			IsHillClimbEnabled:            params.IsHillClimbEnabled,
			ExcludedPools:                 params.ExcludedPools,
			ClientId:                      params.ClientId,
			ExtraFee:                      valueobject.ZeroExtraFee,
			KyberLimitOrderAllowedSenders: params.KyberLimitOrderAllowedSenders,
			IsScaleHelperClient:           params.IsScaleHelperClient,
		}

		if lastSwapState != nil {
			// apply last state before finding route
			for pid, p := range lastSwapState.UpdatedBalancePools {
				state.Pools[pid] = p
			}
			for pid, p := range lastSwapState.UpdatedSwapLimits {
				state.SwapLimit[pid] = p
			}
		}
		findRouteParams := ConvertToPathfinderParams(
			whitelistedTokenSet,
			&pairParams,
			tokenByAddress,
			priceByAddress,
			state,
		)
		findRouteParams.SkipMergeSwap = !a.config.FeatureFlags.IsMergeDuplicateSwapEnabled || params.IsScaleHelperClient

		result, err := a.finderEngine.Find(ctx, findRouteParams)

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
		if result.GetBestRoute() == nil {
			return nil, errors.WithMessagef(ErrRouteNotFound, "bet route is nil")
		}

		route := result.GetBestRoute()
		finalizeExtra, ok := route.ExtraFinalizerData.(types.FinalizeExtraData)
		if !ok {
			logger.Errorf(ctx, "invalid finalizer data %v: %v", pair, route.ExtraFinalizerData)
			return nil, ErrInvalidFinalizerExtraData
		}
		lastSwapState = &finalizeExtra.StateAfterSwap

		allRoutes = append(allRoutes, ConvertToRouteSummary(&pairParams, route))
	}

	return allRoutes, nil
}
