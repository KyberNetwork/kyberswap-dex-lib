package getroute

import (
	"context"
	"math/big"
	"sync"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// aggregator finds best route within amm liquidity sources
type aggregator struct {
	poolRankRepository IPoolRankRepository
	tokenRepository    ITokenRepository
	priceRepository    IPriceRepository
	poolManager        IPoolManager
	bestPathRepository IBestPathRepository

	routeFinder findroute.IFinder

	config AggregatorConfig

	mu sync.RWMutex
}

func NewAggregator(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	poolManager IPoolManager,
	config AggregatorConfig,
	bestPathRepository IBestPathRepository,
) *aggregator {
	routeFinder := spfav2.NewSPFAv2Finder(
		config.FinderOptions.MaxHops,
		config.FinderOptions.DistributionPercent,
		config.FinderOptions.MaxPathsInRoute,
		config.FinderOptions.MaxPathsToGenerate,
		config.FinderOptions.MaxPathsToReturn,
		config.FinderOptions.MinPartUSD,
		config.FinderOptions.MinThresholdAmountInUSD,
		config.FinderOptions.MaxThresholdAmountInUSD,
		bestPathRepository.GetBestPaths,
	)

	return &aggregator{
		poolRankRepository: poolRankRepository,
		tokenRepository:    tokenRepository,
		priceRepository:    priceRepository,
		poolManager:        poolManager,
		routeFinder:        routeFinder,
		config:             config,
		bestPathRepository: bestPathRepository,
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
		stateRoot, err = aevmClient.LatestStateRoot()
		if err != nil {
			return nil, err
		}
	}
	poolByAddress, err := a.getPoolByAddress(ctx, params, common.Hash(stateRoot))
	if err != nil {
		return nil, err
	}

	if len(poolByAddress) == 0 {
		return nil, ErrPoolSetEmpty
	}

	// Step 2: collect tokens and price data
	tokenAddresses := collectTokenAddresses(
		poolByAddress,
		params.TokenIn.Address,
		params.TokenOut.Address,
		params.GasToken.Address,
	)

	tokenByAddress, err := a.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceByAddress, err := a.getPriceUSDByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	// Step 3: finds best route
	return a.findBestRoute(ctx, params, poolByAddress, tokenByAddress, priceByAddress)
}

func (a *aggregator) ApplyConfig(config Config) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.config.FinderOptions != config.Aggregator.FinderOptions || a.config.FeatureFlags != config.Aggregator.FeatureFlags {
		a.routeFinder = spfav2.NewSPFAv2Finder(
			config.Aggregator.FinderOptions.MaxHops,
			config.Aggregator.FinderOptions.DistributionPercent,
			config.Aggregator.FinderOptions.MaxPathsInRoute,
			config.Aggregator.FinderOptions.MaxPathsToGenerate,
			config.Aggregator.FinderOptions.MaxPathsToReturn,
			config.Aggregator.FinderOptions.MinPartUSD,
			config.Aggregator.FinderOptions.MinThresholdAmountInUSD,
			config.Aggregator.FinderOptions.MaxThresholdAmountInUSD,
			a.bestPathRepository.GetBestPaths,
		)
	}

	a.config = config.Aggregator
}

// findBestRoute find the best route and summarize it
func (a *aggregator) findBestRoute(
	ctx context.Context,
	params *types.AggregateParams,
	poolByAddress map[string]poolpkg.IPoolSimulator,
	tokenByAddress map[string]entity.Token,
	priceUSDByAddress map[string]float64,
) (*valueobject.RouteSummary, error) {
	input := findroute.Input{
		TokenInAddress:         params.TokenIn.Address,
		TokenOutAddress:        params.TokenOut.Address,
		AmountIn:               params.AmountIn,
		GasPrice:               params.GasPrice,
		GasTokenPriceUSD:       params.GasTokenPriceUSD,
		SaveGas:                params.SaveGas,
		GasInclude:             params.GasInclude,
		IsPathGeneratorEnabled: params.IsPathGeneratorEnabled,
		SourceHash:             valueobject.HashSources(params.Sources),
	}

	data := findroute.FinderData{
		PoolBucket:        valueobject.NewPoolBucket(poolByAddress),
		TokenByAddress:    tokenByAddress,
		PriceUSDByAddress: priceUSDByAddress,
	}

	routes, err := a.routeFinder.Find(ctx, input, data)
	if err != nil {
		return nil, errors.Wrapf(ErrRouteNotFound, "find route failed: [%v]", err)
	}

	bestRoute := extractBestRoute(routes)

	if bestRoute == nil || len(bestRoute.Paths) == 0 {
		return nil, ErrRouteNotFound
	}

	return a.summarizeRoute(ctx, bestRoute, params, poolByAddress)
}

func (a *aggregator) summarizeRoute(
	_ context.Context,
	route *valueobject.Route,
	params *types.AggregateParams,
	poolByAddress map[string]poolpkg.IPoolSimulator,
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
		tokenAmountIn := path.Input

		for swapIdx, swapPoolAddress := range path.PoolAddresses {
			// Step 2.1.1: take the pool with fresh data
			pool, ok := poolBucket.GetPool(swapPoolAddress)
			if !ok {
				return nil, errors.Wrapf(
					ErrInvalidSwap,
					"aggregator.summarizeRoute > pool not found [%s]",
					swapPoolAddress,
				)
			}

			// Step 2.1.2: simulate c swap through the pool
			result, err := poolpkg.CalcAmountOut(pool, tokenAmountIn, path.Tokens[swapIdx+1].Address)
			if err != nil {
				return nil, errors.Wrapf(
					ErrInvalidSwap,
					"aggregator.summarizeRoute > swap failed > pool: [%s] > error : [%v]",
					swapPoolAddress,
					err,
				)
			}

			// Step 2.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.Wrapf(
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

			metrics.IncrDexHitRate(string(swap.Exchange))
			metrics.IncrPoolTypeHitRate(swap.PoolType)
		}

		// Step 2.2: add up amountOut
		amountOut.Add(amountOut, tokenAmountIn.Amount)
		summarizedRoute = append(summarizedRoute, summarizedPath)
	}

	metrics.IncrRequestPairCount(params.TokenIn.Address, params.TokenOut.Address)

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
	}, nil
}

func (a *aggregator) getPoolByAddress(
	ctx context.Context,
	params *types.AggregateParams,
	stateRoot common.Hash,
) (map[string]poolpkg.IPoolSimulator, error) {
	bestPoolIDs, err := a.poolRankRepository.FindBestPoolIDs(
		ctx,
		params.TokenIn.Address,
		params.TokenOut.Address,
		a.config.GetBestPoolsOptions,
	)
	if err != nil {
		return nil, err
	}

	filteredPoolIDs := make([]string, 0, len(bestPoolIDs))
	for _, bestPoolID := range bestPoolIDs {
		if params.ExcludedPools != nil && params.ExcludedPools.Contains(bestPoolID) {
			continue
		}
		filteredPoolIDs = append(filteredPoolIDs, bestPoolID)
	}

	return a.poolManager.GetPoolByAddress(
		ctx,
		filteredPoolIDs,
		params.Sources,
		stateRoot,
	)
}

// getTokenByAddress receives a list of address and returns a map of address to entity.Token
func (a *aggregator) getTokenByAddress(ctx context.Context, tokenAddresses []string) (map[string]entity.Token, error) {
	tokens, err := a.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = *token
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
