package getroute

import (
	"cmp"
	"context"
	"fmt"
	"math/big"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

// cache is a decorator for aggregator which handle cache logic
type cache struct {
	aggregator IAggregator

	routeCacheRepository IRouteCacheRepository
	poolManager          IPoolManager

	config               valueobject.CacheConfig
	keyGenerator         *routeKeyGenerator
	safetyQuoteReduction *SafetyQuoteReduction
}

func NewCache(
	aggregator IAggregator,
	routeCacheRepository IRouteCacheRepository,
	poolManager IPoolManager,
	config valueobject.CacheConfig,
	safetyQuoteReduction *SafetyQuoteReduction,
) *cache {
	return &cache{
		aggregator:           aggregator,
		routeCacheRepository: routeCacheRepository,
		poolManager:          poolManager,
		config:               config,
		keyGenerator:         newCacheKeyGenerator(config),
		safetyQuoteReduction: safetyQuoteReduction,
	}
}

func (c *cache) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error) {
	var (
		routeSummary *valueobject.RouteSummary
		keys         []valueobject.RouteCacheKeyTTL
		err          error
	)

	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] cache.Aggregate")
	defer span.End()

	res, err := c.keyGenerator.genKey(ctx, params)

	// if this tokenIn has price and we successfully gen cache key
	if err == nil && !res.IsEmpty() {
		keys = res.ToSlice()
		routeSummary, err = c.getRouteFromCache(ctx, params, keys)
		if err == nil {
			return routeSummary, nil
		}
		routeSummary, err = c.aggregator.Aggregate(ctx, params)
		if err != nil {
			return nil, err
		}
		if routeSummary.GetPriceImpact() <= c.config.PriceImpactThreshold {
			c.setRouteToCache(ctx, routeSummary, keys)
		}
	} else {
		// we have no key cacheRoute -> recalculate new route.
		routeSummary, err = c.aggregator.Aggregate(ctx, params)
		if err != nil {
			return nil, err
		}
	}

	return routeSummary, nil
}

func (c *cache) ApplyConfig(config Config) {
	c.keyGenerator.applyConfig(config)
	c.aggregator.ApplyConfig(config)
}

func (c *cache) getBestRouteFromCache(ctx context.Context,
	params *types.AggregateParams,
	keys []valueobject.RouteCacheKeyTTL) (*valueobject.RouteCacheKeyTTL, *valueobject.SimpleRoute, error) {
	cachedRoutes, err := c.routeCacheRepository.Get(ctx, keys)

	if err != nil {
		return nil, nil, err
	}

	if len(cachedRoutes) == 0 {
		return nil, nil, fmt.Errorf("could not find any routes from cache")
	}

	// Compare amount-in to get the best route from Redis cache
	var bestRoute *valueobject.SimpleRoute
	var bestKey valueobject.RouteCacheKeyTTL
	var minDiff *big.Float
	amountInWithoutDecimal := business.AmountWithoutDecimals(params.AmountIn, params.TokenIn.Decimals)
	for key, route := range cachedRoutes {
		if amountInKey, ok := new(big.Float).SetString(key.Key.AmountIn); !ok {
			logger.
				WithFields(ctx, logger.Fields{
					"key":        key.Key.String(""),
					"amountIn":   key.Key.AmountIn,
					"request_id": requestid.GetRequestIDFromCtx(ctx),
				}).
				Info("getBestRouteFromCache Amount in is not a float")
			continue
		} else {
			diff := new(big.Float).Sub(amountInKey, amountInWithoutDecimal)
			diff = diff.Abs(diff)
			if minDiff == nil || diff.Cmp(minDiff) < 0 {
				minDiff = diff
				bestRoute = route
				bestKey = key
			}
		}
	}

	// return error if we can not find bestRoute
	if bestRoute == nil {
		return nil, nil, fmt.Errorf("could not find best routes")
	}

	return &bestKey, bestRoute, nil
}

func (c *cache) getRouteFromCache(ctx context.Context,
	params *types.AggregateParams,
	keys []valueobject.RouteCacheKeyTTL) (*valueobject.RouteSummary, error) {
	bestKey, bestRoute, err := c.getBestRouteFromCache(ctx, params, keys)
	if err != nil {
		logger.
			WithFields(ctx, logger.Fields{
				"key":        keys,
				"reason":     "get cache failed",
				"error":      err,
				"request_id": requestid.GetRequestIDFromCtx(ctx),
				"client_id":  clientid.GetClientIDFromCtx(ctx),
			}).
			Debug("cache missed")
		metrics.IncrFindRouteCacheCount(ctx, false, map[string]string{"reason": "getCachedRouteFailed"})

		return nil, err
	}

	routeSummary, err := c.summarizeSimpleRoute(ctx, bestRoute, params)
	if err != nil {
		logger.
			WithFields(ctx, logger.Fields{
				"key":        bestKey.Key.String(""),
				"reason":     "summarize simple route failed",
				"error":      err,
				"request_id": requestid.GetRequestIDFromCtx(ctx),
				"client_id":  clientid.GetClientIDFromCtx(ctx),
			}).
			Debug("cache missed")
		metrics.IncrFindRouteCacheCount(ctx, false, map[string]string{"reason": "summarizeCachedRouteFailed"})

		return nil, err
	}

	priceImpact := routeSummary.GetPriceImpact()

	if priceImpact > c.config.PriceImpactThreshold {
		logger.
			WithFields(ctx, logger.Fields{
				"key":        bestKey.Key.String(""),
				"reason":     "price impact is greater than threshold",
				"error":      err,
				"request_id": requestid.GetRequestIDFromCtx(ctx),
				"client_id":  clientid.GetClientIDFromCtx(ctx),
			}).
			Debug("cache missed")
		metrics.IncrFindRouteCacheCount(
			ctx,
			false,
			map[string]string{
				"reason": "priceImpactIsGreaterThanEpsilon",
			},
		)

		// it's meaningless to keep a route which cannot be used
		// when we round a get-route input to multiple points, we need to delete multiple cached points if they are useless as well
		// but sometimes, we might delete others useless points which are cached by another input if 2 inputs are overlapse,
		// for simplicity implementation and performance improvement, we might accept this usecase
		c.routeCacheRepository.Del(ctx, keys)
		return nil, errors.WithMessagef(
			ErrPriceImpactIsGreaterThanThreshold,
			"priceImpact: [%f]",
			priceImpact,
		)
	}

	logger.
		WithFields(ctx, logger.Fields{
			"key":        bestKey.Key.String(""),
			"request_id": requestid.GetRequestIDFromCtx(ctx),
			"client_id":  clientid.GetClientIDFromCtx(ctx),
		}).
		Debug("cache hit")
	metrics.IncrFindRouteCacheCount(ctx, true, nil)

	return routeSummary, nil
}

func (c *cache) setRouteToCache(ctx context.Context, routeSummary *valueobject.RouteSummary, keys []valueobject.RouteCacheKeyTTL) {
	simpleRoute := simplifyRouteSummary(routeSummary)
	routes := make([]*valueobject.SimpleRoute, 0, len(keys))
	for range keys {
		routes = append(routes, simpleRoute)
	}

	if err := c.routeCacheRepository.Set(ctx, keys, routes); err != nil {
		logger.
			WithFields(ctx, logger.Fields{"error": err}).
			Error("cache.setRouteToCache failed")
	}
}

func setToSlice[T cmp.Ordered](set mapset.Set[T]) []T {
	if set == nil {
		return nil
	}
	return mapset.Sorted(set)
}

func (c *cache) summarizeSimpleRoute(
	ctx context.Context,
	simpleRoute *valueobject.SimpleRoute,
	params *types.AggregateParams,
) (*valueobject.RouteSummary, error) {
	// Step 1: prepare pool data
	var (
		stateRoot aevmcommon.Hash
		err       error
	)
	if aevmClient := c.poolManager.GetAEVMClient(); aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot(ctx)
		if err != nil {
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}
	state, err := c.poolManager.GetStateByPoolAddresses(
		ctx,
		simpleRoute.ExtractPoolAddresses(),
		params.Sources,
		common.Hash(stateRoot),
	)
	if err != nil {
		return nil, err
	}

	poolBucket := valueobject.NewPoolBucket(state.Pools)
	var (
		amountOut = new(big.Int).Set(constant.Zero)
		gas       = business.BaseGas
	)

	// Step 2: distribute amountIn into paths following distributions
	distributedAmounts := business.DistributeAmount(params.AmountIn, simpleRoute.Distributions)

	// Step 3: summarize route
	summarizedRoute := make([][]valueobject.Swap, 0, len(simpleRoute.Paths))
	for pathIdx, simplePath := range simpleRoute.Paths {

		// Step 3.1: summarize path
		summarizedPath := make([]valueobject.Swap, 0, len(simplePath))

		// Step 3.1.0: prepare input of the first swap
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  simplePath[0].TokenInAddress,
			Amount: distributedAmounts[pathIdx],
		}

		for _, simpleSwap := range simplePath {
			// Step 3.1.1: take the pool with fresh data
			pool, ok := poolBucket.GetPool(simpleSwap.PoolAddress)
			if !ok {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"cache.summarizeSimpleRoute > pool not found [%s]",
					simpleSwap.PoolAddress,
				)
			}

			swapLimit := state.SwapLimit[pool.GetType()]
			// Step 3.1.2: simulate c swap through the pool
			result, err := routerpoolpkg.CalcAmountOut(ctx, pool, tokenAmountIn, simpleSwap.TokenOutAddress, swapLimit)
			if err != nil {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"cache.summarizeSimpleRoute > swap failed > pool: [%s] > error : [%v]",
					simpleSwap.PoolAddress,
					err,
				)
			}

			// Step 3.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"cache.summarizeSimpleRoute > invalid swap > pool : [%s]",
					simpleSwap.PoolAddress,
				)
			}

			// Step 3.1.4: update balance of the pool
			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
				SwapLimit:      swapLimit,
			}
			pool = poolBucket.ClonePool(simpleSwap.PoolAddress)
			pool.UpdateBalance(updateBalanceParams)

			// Step 3.1.5
			// We need to calculate safety quoting amount and reasign new amount out to next path's amount in
			reducedNextAmountIn := c.safetyQuoteReduction.Reduce(
				result.TokenAmountOut,
				c.safetyQuoteReduction.GetSafetyQuotingRate(pool.GetType()),
				params.ClientId)

			// Step 3.1.6: summarize the swap
			// important: must re-update amount out to reducedNextAmountIn
			swap := valueobject.Swap{
				Pool:              simpleSwap.PoolAddress,
				TokenIn:           simpleSwap.TokenInAddress,
				TokenOut:          simpleSwap.TokenOutAddress,
				SwapAmount:        tokenAmountIn.Amount,
				AmountOut:         reducedNextAmountIn.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(pool.GetExchange()),
				PoolLength:        len(pool.GetTokens()),
				PoolType:          pool.GetType(),
				PoolExtra:         pool.GetMetaInfo(simpleSwap.TokenInAddress, simpleSwap.TokenOutAddress),
				Extra:             result.SwapInfo,
			}

			summarizedPath = append(summarizedPath, swap)

			// Step 3.1.7: add up gas fee
			gas += result.Gas

			// Step 3.1.8: update input of the next swap is output of current swap
			tokenAmountIn = reducedNextAmountIn
		}

		// Step 3.2: add up amountOut
		amountOut.Add(amountOut, tokenAmountIn.Amount)
		summarizedRoute = append(summarizedRoute, summarizedPath)
	}

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
