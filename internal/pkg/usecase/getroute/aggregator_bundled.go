package getroute

import (
	"context"
	"fmt"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type bundledAggregator struct {
	*aggregator
}

func NewBundledAggregator(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	onchainpriceRepository IOnchainPriceRepository,
	poolManager IPoolManager,
	config AggregatorConfig,
	finderEngine finderEngine.IPathFinderEngine,
) *bundledAggregator {
	ag := &aggregator{
		poolRankRepository:     poolRankRepository,
		tokenRepository:        tokenRepository,
		priceRepository:        priceRepository,
		onchainpriceRepository: onchainpriceRepository,
		poolManager:            poolManager,
		finderEngine:           finderEngine,
		config:                 config,
	}
	return &bundledAggregator{ag}
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

	state, err := a.getStateByBundledAddress(ctx, params, common.Hash(stateRoot))
	if err != nil {
		return nil, err
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

	var priceUSDByAddress map[string]float64

	// only get price from onchain-price-service if enabled
	var priceByAddress map[string]*routerEntity.OnchainPrice
	if a.onchainpriceRepository != nil {
		priceByAddress, err = a.onchainpriceRepository.FindByAddresses(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}
	} else {
		priceUSDByAddress, err = a.getPriceUSDByAddress(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}
	}

	// Step 3: finds best route
	return a.findBestBundledRoute(ctx, params, tokenByAddress, priceUSDByAddress, priceByAddress, state)
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
			ctx,
			pair.TokenIn,
			pair.TokenOut,
			a.config.GetBestPoolsOptions,
		)
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
	)
}

func (a *bundledAggregator) findBestBundledRoute(
	ctx context.Context,
	params *types.AggregateBundledParams,
	tokenByAddress map[string]*entity.Token,
	priceUSDByAddress map[string]float64,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
) ([]*valueobject.RouteSummary, error) {
	allRoutes := make([]*valueobject.RouteSummary, 0, len(params.Pairs))

	var lastSwapState *types.StateAfterSwap

	gasToken, ok := tokenByAddress[params.GasToken]
	if !ok {
		return nil, errors.WithMessagef(ErrInvalidToken, "invalid gasToken: %v", params.GasToken)
	}
	gasTokenPrice := GetPriceOnchainWithFallback(params.GasToken, priceUSDByAddress, priceByAddress, true)

	for _, pair := range params.Pairs {
		tokenIn, ok := tokenByAddress[pair.TokenIn]
		if !ok {
			return nil, errors.WithMessagef(ErrInvalidToken, "invalid tokenIn: %v", pair.TokenIn)
		}
		tokenOut, ok := tokenByAddress[pair.TokenOut]
		if !ok {
			return nil, errors.WithMessagef(ErrInvalidToken, "invalid tokenOut: %v", pair.TokenOut)
		}
		tokenInPrice := GetPriceOnchainWithFallback(pair.TokenIn, priceUSDByAddress, priceByAddress, false)  // use sell price for tokenIn
		tokenOutPrice := GetPriceOnchainWithFallback(pair.TokenOut, priceUSDByAddress, priceByAddress, true) // use buy price for token out and gas

		amountInUSD := utils.CalcTokenAmountUsd(pair.AmountIn, tokenIn.Decimals, tokenInPrice)
		if amountInUSD > MaxAmountInUSD {
			return nil, ErrAmountInIsGreaterThanMaxAllowed
		}

		pairParams := types.AggregateParams{
			TokenIn:            *tokenIn,
			TokenOut:           *tokenOut,
			GasToken:           *gasToken,
			TokenInPriceUSD:    tokenInPrice,
			TokenOutPriceUSD:   tokenOutPrice,
			GasTokenPriceUSD:   gasTokenPrice,
			AmountIn:           pair.AmountIn,
			Sources:            params.Sources,
			SaveGas:            params.SaveGas,
			GasInclude:         params.GasInclude,
			GasPrice:           params.GasPrice,
			IsHillClimbEnabled: params.IsHillClimbEnabled,
			ExcludedPools:      params.ExcludedPools,
			ClientId:           params.ClientId,
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
			a.config.WhitelistedTokenSet,
			&pairParams,
			tokenByAddress,
			priceUSDByAddress,
			priceByAddress,
			state,
		)

		route, err := a.finderEngine.Find(ctx, findRouteParams)

		if err != nil {
			if errors.Is(err, finderEngine.ErrInvalidSwap) {
				return nil, errors.WithMessagef(ErrInvalidSwap, "find route failed: [%v]", err)
			} else if errors.Is(err, finderEngine.ErrRouteNotFound) {
				return nil, errors.WithMessagef(ErrRouteNotFound, "find route failed: [%v]", err)
			} else {
				return nil, err
			}
		}

		lastSwapState, ok = route.ExtraFinalizerData.(*types.StateAfterSwap)
		if !ok {
			logger.Errorf(ctx, "invalid finalizer data %v: %v", pair, route.ExtraFinalizerData)
			return nil, ErrInvalidFinalizerExtraData
		}

		allRoutes = append(allRoutes, ConvertToRouteSummary(&pairParams, route))
	}

	return allRoutes, nil
}
