package getroute

import (
	"cmp"
	"context"
	"fmt"
	"math/big"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finalizer"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
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
	finderEngine finderEngine.IPathFinderEngine
}

func NewCache(
	aggregator IAggregator,
	routeCacheRepository IRouteCacheRepository,
	poolManager IPoolManager,
	config valueobject.CacheConfig,
	finderEngine finderEngine.IPathFinderEngine,
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

func (c *cache) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummaries, error) {
	var (
		routeSummaries *valueobject.RouteSummaries
		keys           []valueobject.RouteCacheKeyTTL
		err            error
	)

	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] cache.Aggregate")
	defer span.End()

	res, err := c.keyGenerator.genKey(ctx, params)

	// if this tokenIn has price and we successfully gen cache key
	if err == nil && !res.IsEmpty() {
		keys = res.ToSlice()
		routeSummaries, err = c.getRouteFromCache(ctx, params, keys)
		if err == nil {
			return routeSummaries, nil
		}
		routeSummaries, err = c.aggregator.Aggregate(ctx, params)
		if err != nil {
			return nil, err
		}

		routeSummary := routeSummaries.GetBestRouteSummary()
		if routeSummary == nil {
			return nil, errors.Errorf("best route is nil")
		}

		// If the amount in USD is nearly insignificant (or 0), price impact is -Inf, so ignore price impact check if cache point is base on amountIn (not amountInUSD)
		// We don't support cache merged swaps route for now.
		// TODO: Improve caching solution to support merged swaps route,
		// and other more general routes that each path may not always
		// start from params.TokenIn -> params.TokenOut.
		if (routeSummary.AmountInUSD < c.config.MinAmountInUSD ||
			routeSummary.GetPriceImpact() <= c.config.PriceImpactThreshold) &&
			!isMergeSwapRoute(params, routeSummary) {
			c.setRouteToCache(ctx, routeSummaries, keys)
		}
	} else {
		// we have no key cacheRoute -> recalculate new route.
		routeSummaries, err = c.aggregator.Aggregate(ctx, params)
		if err != nil {
			return nil, err
		}
	}

	return routeSummaries, nil
}

func (c *cache) ApplyConfig(config Config) {
	c.keyGenerator.applyConfig(config)
	c.aggregator.ApplyConfig(config)
}

func (c *cache) getBestRouteFromCache(ctx context.Context,
	params *types.AggregateParams,
	keys []valueobject.RouteCacheKeyTTL) (*valueobject.RouteCacheKeyTTL, *valueobject.SimpleRouteWithExtraData, error) {
	cachedRoutes, err := c.routeCacheRepository.Get(ctx, keys)

	if err != nil {
		return nil, nil, err
	}

	if len(cachedRoutes) == 0 {
		return nil, nil, fmt.Errorf("could not find any routes from cache")
	}

	// Compare amount-in to get the best route from Redis cache
	var bestRoute *valueobject.SimpleRouteWithExtraData
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
	keys []valueobject.RouteCacheKeyTTL) (*valueobject.RouteSummaries, error) {
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
		metrics.CountFindRouteCache(ctx, false, "reason", "getCachedRouteFailed")

		return nil, err
	}

	var stateRoot aevmcommon.Hash
	if aevmClient := c.poolManager.GetAEVMClient(); aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot(ctx)
		if err != nil {
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}

	routeSummaries, err := c.summarizeSimpleRouteWithExtraData(ctx, bestRoute, params, stateRoot)
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
		metrics.CountFindRouteCache(ctx, false, "reason", "summarizeCachedRouteFailed")
		return nil, err
	}
	routeSummary := routeSummaries.GetBestRouteSummary()

	// don't need to check priceImpact if cache point is base on amountIn (not amountInUSD) because tokenIn has no price
	// GetPriceImpact() will return -Inf if tokenIn has no prices, but keep this check available for explicit logic
	if routeSummary.AmountInUSD < c.config.MinAmountInUSD {
		return routeSummaries, nil
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
		metrics.CountFindRouteCache(ctx, false, "reason", "priceImpactIsGreaterThanEpsilon")

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
	metrics.CountFindRouteCache(ctx, true)

	return routeSummaries, nil
}

func (c *cache) setRouteToCache(ctx context.Context, routeSummaries *valueobject.RouteSummaries, keys []valueobject.RouteCacheKeyTTL) {
	bestSimpleRoute := simplifyRouteSummary(routeSummaries.GetBestRouteSummary())
	route := valueobject.SimpleRouteWithExtraData{BestRoute: bestSimpleRoute}
	if routeSummaries.GetAMMBestRouteSummary() != nil {
		route.AMMRoute = simplifyRouteSummary(routeSummaries.GetAMMBestRouteSummary())
	}
	routes := make([]*valueobject.SimpleRouteWithExtraData, 0, len(keys))
	for range keys {
		routes = append(routes, &route)
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
	finalizer finderEngine.IFinalizer,
	stateRoot aevmcommon.Hash,
) (*finderEntity.Route, error) {
	// Step 1: prepare pool data
	var err error
	poolAddresses := simpleRoute.ExtractPoolAddresses()
	state, err := c.poolManager.GetStateByPoolAddresses(
		ctx,
		poolAddresses,
		params.Sources,
		common.Hash(stateRoot),
		types.PoolManagerExtraData{
			KyberLimitOrderAllowedSenders: params.KyberLimitOrderAllowedSenders,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(state.Pools) != len(poolAddresses) {
		return nil, errors.New("could not get all pools from pool manager")
	}
	constructRoute := c.convertSimpleRouteToConstructRoute(simpleRoute, params)

	tokenByAddress := map[string]*entity.Token{
		params.TokenIn.Address:  &params.TokenIn,
		params.TokenOut.Address: &params.TokenOut,
		params.GasToken.Address: &params.GasToken,
	}

	priceByAddress := map[string]*routerEntity.OnchainPrice{
		params.TokenIn.Address: {
			USDPrice: routerEntity.Price{
				Buy: big.NewFloat(params.TokenInPriceUSD),
			},
		},
		params.TokenOut.Address: {
			USDPrice: routerEntity.Price{
				Buy: big.NewFloat(params.TokenOutPriceUSD),
			},
		},
		params.GasToken.Address: {
			USDPrice: routerEntity.Price{
				Buy: big.NewFloat(params.GasTokenPriceUSD),
			},
		},
	}

	findRouteParams := ConvertToPathfinderParams(
		nil,
		params,
		tokenByAddress,
		priceByAddress,
		state,
	)

	route, err := finalizer.Finalize(ctx, findRouteParams, constructRoute, nil)
	if err != nil {
		return nil, err
	}

	return route, nil
}

func (c *cache) summarizeSimpleRouteWithExtraData(
	ctx context.Context,
	simpleRoute *valueobject.SimpleRouteWithExtraData,
	params *types.AggregateParams,
	stateRoot aevmcommon.Hash,
) (*valueobject.RouteSummaries, error) {
	var err error

	// If route contains AMM best route, then summarize AMM best route first
	var ammRoute *finderEntity.Route
	if simpleRoute.AMMRoute != nil {
		ammRoute, err = c.summarizeSimpleRoute(
			ctx, simpleRoute.AMMRoute, params, finalizer.NewDefaultFinalizer(), stateRoot)
		if err != nil {
			logger.
				WithFields(ctx, logger.Fields{
					"error":      err,
					"request_id": requestid.GetRequestIDFromCtx(ctx),
					"client_id":  clientid.GetClientIDFromCtx(ctx),
				}).
				Warnf("summarize ammRoute failed")
			// TODO: count metric
		}
	}

	// Step 1: prepare pool data
	poolAddresses := simpleRoute.BestRoute.ExtractPoolAddresses()
	state, err := c.poolManager.GetStateByPoolAddresses(
		ctx,
		poolAddresses,
		params.Sources,
		common.Hash(stateRoot),
		types.PoolManagerExtraData{
			KyberLimitOrderAllowedSenders: params.KyberLimitOrderAllowedSenders,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(state.Pools) != len(poolAddresses) {
		return nil, errors.New("could not get all pools from pool manager")
	}
	constructRoute := c.convertSimpleRouteToConstructRoute(simpleRoute.BestRoute, params)
	ammConstructRoute := c.convertRouteToConstructRoute(ammRoute, params)
	bestRouteResult := finderCommon.BestRouteResult{
		BestRoutes:   []*finderCommon.ConstructRoute{constructRoute},
		AMMBestRoute: ammConstructRoute,
	}

	tokenByAddress := map[string]*entity.Token{
		params.TokenIn.Address:  &params.TokenIn,
		params.TokenOut.Address: &params.TokenOut,
		params.GasToken.Address: &params.GasToken,
	}

	priceByAddress := map[string]*routerEntity.OnchainPrice{
		params.TokenIn.Address: {
			USDPrice: routerEntity.Price{
				Buy: big.NewFloat(params.TokenInPriceUSD),
			},
		},
		params.TokenOut.Address: {
			USDPrice: routerEntity.Price{
				Buy: big.NewFloat(params.TokenOutPriceUSD),
			},
		},
		params.GasToken.Address: {
			USDPrice: routerEntity.Price{
				Buy: big.NewFloat(params.GasTokenPriceUSD),
			},
		},
	}

	findRouteParams := ConvertToPathfinderParams(
		nil,
		params,
		tokenByAddress,
		priceByAddress,
		state,
	)

	finalizer := c.finderEngine.GetFinalizer()
	route, err := finalizer.Finalize(ctx, findRouteParams, constructRoute, finalizer.GetExtraData(ctx, &bestRouteResult))
	if err != nil {
		return nil, err
	}

	return &valueobject.RouteSummaries{ConvertToRouteSummary(params, route), ConvertToRouteSummary(params, ammRoute)}, nil
}

func (c *cache) convertSimpleRouteToConstructRoute(simpleRoute *valueobject.SimpleRoute, params *types.AggregateParams) *finderCommon.ConstructRoute {
	distributedAmounts := business.DistributeAmount(params.AmountIn, simpleRoute.Distributions)

	constructRoute := finderCommon.NewConstructRoute(params.TokenIn.Address, params.TokenOut.Address, c.finderEngine.GetFinder().CustomFuncs())
	for pathIdx, simplePath := range simpleRoute.Paths {
		constructPath := finderCommon.NewConstructPath(distributedAmounts[pathIdx], c.finderEngine.GetFinder().CustomFuncs())
		constructPath.AddToken(params.TokenIn.Address)

		for _, simpleSwap := range simplePath {
			constructPath.AddPool(simpleSwap.PoolAddress)
			constructPath.AddToken(simpleSwap.TokenOutAddress)
		}

		constructRoute.AddPath(constructPath)
	}

	return constructRoute
}

func (c *cache) convertRouteToConstructRoute(route *finderEntity.Route, params *types.AggregateParams) *finderCommon.ConstructRoute {
	if route == nil {
		return nil
	}

	constructRoute := finderCommon.NewConstructRoute(params.TokenIn.Address, params.TokenOut.Address, c.finderEngine.GetFinder().CustomFuncs())

	for _, swaps := range route.Route {
		constructPath := finderCommon.NewConstructPath(swaps[0].SwapAmount, c.finderEngine.GetFinder().CustomFuncs())
		constructPath.AddToken(params.TokenIn.Address)

		for _, swap := range swaps {
			constructPath.AddPool(swap.Pool)
			constructPath.AddToken(swap.TokenOut)
		}

		constructRoute.AddPath(constructPath)
	}

	return constructRoute
}

func isMergeSwapRoute(params *types.AggregateParams, routeSummary *valueobject.RouteSummary) bool {
	for _, path := range routeSummary.Route {
		tokenIn := path[0].TokenIn
		tokenOut := path[len(path)-1].TokenOut
		if tokenIn != params.TokenIn.Address || tokenOut != params.TokenOut.Address {
			return true
		}
	}

	return false
}
