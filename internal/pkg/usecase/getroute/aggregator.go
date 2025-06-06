package getroute

import (
	"context"
	"fmt"
	"sync"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

// aggregator finds best route within amm liquidity sources
type aggregator struct {
	poolRankRepository     IPoolRankRepository
	tokenRepository        ITokenRepository
	onchainpriceRepository IOnchainPriceRepository
	poolManager            IPoolManager

	finderEngine finderEngine.IPathFinderEngine

	config AggregatorConfig
	mu     sync.RWMutex
}

func NewAggregator(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	onchainpriceRepository IOnchainPriceRepository,
	poolManager IPoolManager,
	config AggregatorConfig,
	finderEngine finderEngine.IPathFinderEngine,
) *aggregator {
	return &aggregator{
		poolRankRepository:     poolRankRepository,
		tokenRepository:        tokenRepository,
		onchainpriceRepository: onchainpriceRepository,
		poolManager:            poolManager,
		finderEngine:           finderEngine,
		config:                 config,
	}
}

func (a *aggregator) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummaries,
	error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] aggregator.Aggregate")
	defer span.End()

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

	state, err := a.getStateByAddress(ctx, params, common.Hash(stateRoot))
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
	return a.findBestRoute(ctx, params, tokenByAddress, priceByAddress, state)
}

func (a *aggregator) ApplyConfig(config Config) {
	a.mu.Lock()
	defer a.mu.Unlock()

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
	)
	findRouteParams.SkipMergeSwap = !a.config.FeatureFlags.IsMergeDuplicateSwapEnabled || params.IsScaleHelperClient

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

func (a *aggregator) getStateByAddress(
	ctx context.Context,
	params *types.AggregateParams,
	stateRoot common.Hash,
) (*types.FindRouteState, error) {
	if len(params.Sources) == 0 {
		logger.Errorf(ctx, "sources list is empty, error: %v", ErrPoolSetFiltered)
		return nil, ErrPoolSetFiltered
	}
	bestPoolIDs, err := a.poolRankRepository.FindBestPoolIDs(ctx, params.TokenIn.Address, params.TokenOut.Address,
		params.AmountInUsd, a.config.GetBestPoolsOptions, params.Index, params.ForcePoolsForToken)

	if err != nil {
		return nil, err
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
		logger.Errorf(ctx, "empty filtered pool IDs. bestPoolIDs %v, excludedPools: %v, returning error: %v",
			bestPoolIDs, params.ExcludedPools.String(), ErrPoolSetFiltered)
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
