package getroute

import (
	"cmp"
	"context"
	"fmt"
	"math/big"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
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

	config       valueobject.CacheConfig
	keyGenerator *routeKeyGenerator
	finderEngine findroute.IFinderEngine
}

func NewCache(
	aggregator IAggregator,
	routeCacheRepository IRouteCacheRepository,
	poolManager IPoolManager,
	config valueobject.CacheConfig,
	finderEngine findroute.IFinderEngine,
) *cache {
	return &cache{
		aggregator:           aggregator,
		routeCacheRepository: routeCacheRepository,
		poolManager:          poolManager,
		config:               config,
		keyGenerator:         newCacheKeyGenerator(config),
		finderEngine:         finderEngine,
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
		// If the amount in USD is nearly insignificant (or 0), price impact is -Inf, so ignore price impact check if cache point is base on amountIn (not amountInUSD)
		if routeSummary.AmountInUSD < c.config.MinAmountInUSD || routeSummary.GetPriceImpact() <= c.config.PriceImpactThreshold {
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

	// don't need to check priceImpact if cache point is base on amountIn (not amountInUSD) because tokenIn has no price
	// GetPriceImpact() will return -Inf if tokenIn has no prices, but keep this check available for explicit logic
	if routeSummary.AmountInUSD < c.config.MinAmountInUSD {
		return routeSummary, nil
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

	return c.finderEngine.GetFinalizer().FinalizeSimpleRoute(ctx, simpleRoute, state.Pools, state.SwapLimit, params)
}
