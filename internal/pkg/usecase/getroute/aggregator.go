package getroute

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/hillclimb"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

// aggregator finds best route within amm liquidity sources
type aggregator struct {
	poolRankRepository     IPoolRankRepository
	tokenRepository        ITokenRepository
	priceRepository        IPriceRepository
	onchainpriceRepository IOnchainPriceRepository
	poolManager            IPoolManager
	bestPathRepository     IBestPathRepository

	routeFinder          findroute.IFinder
	hillClimbRouteFinder findroute.IFinder
	config               AggregatorConfig

	mu sync.RWMutex
}

func NewAggregator(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	onchainpriceRepository IOnchainPriceRepository,
	poolManager IPoolManager,
	config AggregatorConfig,
	bestPathRepository IBestPathRepository,
	routeFinder findroute.IFinder,
) *aggregator {

	hillClimbRouteFinder := hillclimb.NewHillClimbingFinder(
		config.FinderOptions.HillClimbDistributionPercent,
		config.FinderOptions.HillClimbIteration,
		config.FinderOptions.HillClimbMinPartUSD,
		routeFinder,
	)

	return &aggregator{
		poolRankRepository:     poolRankRepository,
		tokenRepository:        tokenRepository,
		priceRepository:        priceRepository,
		onchainpriceRepository: onchainpriceRepository,
		poolManager:            poolManager,
		routeFinder:            routeFinder,
		hillClimbRouteFinder:   hillClimbRouteFinder,
		config:                 config,
		bestPathRepository:     bestPathRepository,
	}
}

func (a *aggregator) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error) {
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
	tokenAddresses := CollectTokenAddresses(
		state.Pools,
		params.TokenIn.Address,
		params.TokenOut.Address,
		params.GasToken.Address,
	)

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
	return a.findBestRoute(ctx, params, tokenByAddress, priceUSDByAddress, priceByAddress, state)
}

func (a *aggregator) ApplyConfig(config Config) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var getBestPaths func(sourceHash uint64, tokenIn, tokenOut string) []*entity.MinimalPath
	if a.bestPathRepository != nil {
		getBestPaths = a.bestPathRepository.GetBestPaths
	}

	var routeFinder findroute.IFinder = spfav2.NewSPFAv2Finder(
		config.Aggregator.FinderOptions.MaxHops,
		config.Aggregator.WhitelistedTokenSet,
		config.Aggregator.FinderOptions.DistributionPercent,
		config.Aggregator.FinderOptions.MaxPathsInRoute,
		config.Aggregator.FinderOptions.MaxPathsToGenerate,
		config.Aggregator.FinderOptions.MaxPathsToReturn,
		config.Aggregator.FinderOptions.MinPartUSD,
		config.Aggregator.FinderOptions.MinThresholdAmountInUSD,
		config.Aggregator.FinderOptions.MaxThresholdAmountInUSD,
		getBestPaths,
	)

	a.routeFinder = routeFinder
	a.hillClimbRouteFinder = hillclimb.NewHillClimbingFinder(
		config.Aggregator.FinderOptions.HillClimbDistributionPercent,
		config.Aggregator.FinderOptions.HillClimbIteration,
		config.Aggregator.FinderOptions.MinPartUSD,
		routeFinder,
	)

	a.config = config.Aggregator
}

// findBestRoute find the best route and summarize it
func (a *aggregator) findBestRoute(
	ctx context.Context,
	params *types.AggregateParams,
	tokenByAddress map[string]*entity.Token,
	priceUSDByAddress map[string]float64,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
) (*valueobject.RouteSummary, error) {
	input := findroute.Input{
		TokenInAddress:         params.TokenIn.Address,
		TokenOutAddress:        params.TokenOut.Address,
		AmountIn:               params.AmountIn,
		GasPrice:               params.GasPrice,
		GasTokenPriceUSD:       params.GasTokenPriceUSD,
		TokenInPriceUSD:        params.TokenInPriceUSD,
		SaveGas:                params.SaveGas,
		GasInclude:             params.GasInclude,
		IsPathGeneratorEnabled: params.IsPathGeneratorEnabled,
		SourceHash:             valueobject.HashSources(params.Sources),
	}

	data := findroute.NewFinderData(ctx, tokenByAddress, priceUSDByAddress, priceByAddress, state)
	defer data.ReleaseResources()
	var (
		routes []*valueobject.Route
		err    error
	)
	if params.IsHillClimbEnabled {
		routes, err = a.hillClimbRouteFinder.Find(ctx, input, data)
	} else {
		routes, err = a.routeFinder.Find(ctx, input, data)
	}
	if err != nil {
		return nil, errors.WithMessagef(ErrRouteNotFound, "find route failed: [%v]", err)
	}

	bestRoute := extractBestRoute(routes)

	if bestRoute == nil || len(bestRoute.Paths) == 0 {
		return nil, ErrRouteNotFound
	}

	data.Refresh()
	return a.summarizeRoute(ctx, bestRoute, params, state.Pools, data.SwapLimits)
}

func (a *aggregator) summarizeRoute(
	ctx context.Context,
	route *valueobject.Route,
	params *types.AggregateParams,
	poolByAddress map[string]poolpkg.IPoolSimulator,
	swapLimits map[string]poolpkg.SwapLimit,
) (*valueobject.RouteSummary, error) {
	// Step 1: prepare pool data
	poolBucket := valueobject.NewPoolBucket(poolByAddress)

	var (
		amountOut = new(big.Int).Set(constant.Zero)
		gas       = business.BaseGas
	)

	// Step 2: summarize route
	summarizedRoute := make([][]valueobject.Swap, 0, len(route.Paths))
	for _, path := range route.Paths {

		// Step 2.1: summarize path
		summarizedPath := make([]valueobject.Swap, 0, len(path.PoolAddresses))

		// Step 2.1.0: prepare input of the first swap
		tokenAmountIn := *path.Input.ToDexLibAmount()

		for swapIdx, swapPoolAddress := range path.PoolAddresses {
			// Step 2.1.1: take the pool with fresh data
			pool, ok := poolBucket.GetPool(swapPoolAddress)
			if !ok {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"aggregator.summarizeRoute > pool not found [%s]",
					swapPoolAddress,
				)
			}

			swapLimit := swapLimits[pool.GetType()]
			// Step 2.1.2: simulate c swap through the pool
			result, err := routerpoolpkg.CalcAmountOut(ctx, pool, tokenAmountIn, path.Tokens[swapIdx+1].Address, swapLimit)
			if err != nil {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"aggregator.summarizeRoute > swap failed > pool: [%s] > error : [%v]",
					swapPoolAddress,
					err,
				)
			}

			// Step 2.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"aggregator.summarizeRoute > invalid swap > pool : [%s]",
					swapPoolAddress,
				)
			}

			//Step 2.1.4: clone the pool before updating it (do not modify IPool returned by `poolManager`)
			pool = poolBucket.ClonePool(swapPoolAddress)

			// Step 2.1.5: update balance of the pool
			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
				SwapLimit:      swapLimit,
			}
			pool.UpdateBalance(updateBalanceParams)

			// Step 2.1.6: summarize the swap
			swap := valueobject.Swap{
				Pool:              pool.GetAddress(),
				TokenIn:           tokenAmountIn.Token,
				TokenOut:          result.TokenAmountOut.Token,
				SwapAmount:        tokenAmountIn.Amount,
				AmountOut:         result.TokenAmountOut.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(pool.GetExchange()),
				PoolLength:        len(pool.GetTokens()),
				PoolType:          pool.GetType(),
				PoolExtra:         pool.GetMetaInfo(tokenAmountIn.Token, result.TokenAmountOut.Token),
				Extra:             result.SwapInfo,
			}

			summarizedPath = append(summarizedPath, swap)

			// Step 2.1.7: add up gas fee
			gas += result.Gas

			// Step 2.1.8: update input of the next swap is output of current swap
			tokenAmountIn = *result.TokenAmountOut

			metrics.IncrDexHitRate(ctx, string(swap.Exchange))
			metrics.IncrPoolTypeHitRate(ctx, swap.PoolType)
		}

		// Step 2.2: add up amountOut
		amountOut.Add(amountOut, tokenAmountIn.Amount)
		summarizedRoute = append(summarizedRoute, summarizedPath)
	}

	metrics.IncrRequestPairCount(ctx, params.TokenIn.Address, params.TokenOut.Address)

	return &valueobject.RouteSummary{
		TokenIn:      params.TokenIn.Address,
		AmountIn:     params.AmountIn,
		AmountInUSD:  utils.CalcTokenAmountUsd(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD),
		TokenOut:     params.TokenOut.Address,
		AmountOut:    amountOut,
		AmountOutUSD: utils.CalcTokenAmountUsd(amountOut, params.TokenOut.Decimals, params.TokenOutPriceUSD),
		Gas:          gas,
		GasPrice:     params.GasPrice,
		GasUSD:       utils.CalcGasUsd(params.GasPrice, gas, params.GasTokenPriceUSD),
		ExtraFee:     params.ExtraFee,
		Route:        summarizedRoute,
		Extra:        route.Extra,
	}, nil
}

func (a *aggregator) getStateByAddress(
	ctx context.Context,
	params *types.AggregateParams,
	stateRoot common.Hash,
) (*types.FindRouteState, error) {
	if len(params.Sources) == 0 {
		return nil, ErrPoolSetFiltered
	}

	bestPoolIDs, err := a.poolRankRepository.FindBestPoolIDs(
		ctx,
		params.TokenIn.Address,
		params.TokenOut.Address,
		a.config.GetBestPoolsOptions,
	)
	if err != nil {
		return nil, err
	}

	if len(bestPoolIDs) == 0 {
		logger.Error(ctx, "empty bestPoolIDs ")
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

// getPriceUSDByAddress receives a list of address and returns a map of address to its preferred price in USD
func (a *aggregator) getPriceUSDByAddress(ctx context.Context, tokenAddresses []string) (map[string]float64, error) {
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
