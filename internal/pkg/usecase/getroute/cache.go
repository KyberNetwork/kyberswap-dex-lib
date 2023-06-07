package getroute

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"

	"github.com/pkg/errors"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

// cache is a decorator for aggregator which handle cache logic
type cache struct {
	aggregator IAggregator

	routeCacheRepository IRouteCacheRepository
	poolManager          IPoolManager

	shrinkFunc ShrinkFunc

	config CacheConfig

	mu sync.RWMutex
}

func NewCache(
	aggregator IAggregator,
	routeCacheRepository IRouteCacheRepository,
	poolManager IPoolManager,
	config CacheConfig,
) *cache {
	return &cache{
		aggregator:           aggregator,
		routeCacheRepository: routeCacheRepository,
		poolManager:          poolManager,
		shrinkFunc:           ShrinkFuncFactory(config),
		config:               config,
	}
}

func (c *cache) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] cache.Aggregate")
	defer span.Finish()

	key, ttl := c.genKey(params)

	routeSummary, err := c.getRouteFromCache(ctx, params, key)
	if err == nil {
		return routeSummary, nil
	}

	routeSummary, err = c.aggregator.Aggregate(ctx, params)
	if err != nil {
		return nil, err
	}

	if routeSummary.GetPriceImpact() <= c.config.PriceImpactThreshold {
		c.setRouteToCache(ctx, routeSummary, key, ttl)
	}

	return routeSummary, nil
}

func (c *cache) ApplyConfig(config Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = config.Cache
	c.aggregator.ApplyConfig(config)
}

func (c *cache) getRouteFromCache(ctx context.Context, params *types.AggregateParams, key *valueobject.RouteCacheKey) (*valueobject.RouteSummary, error) {
	simpleRoute, err := c.routeCacheRepository.Get(ctx, key)
	if err != nil {
		logger.
			WithFields(logger.Fields{
				"key":        key.String(""),
				"reason":     "get cache failed",
				"error":      err,
				"request_id": requestid.RequestIDFromCtx(ctx),
			}).
			Info("cache missed")
		metrics.IncrFindRouteCacheCount(false, []string{"reason:getCachedRouteFailed"})

		return nil, err
	}

	routeSummary, err := c.summarizeSimpleRoute(ctx, simpleRoute, params)
	if err != nil {
		logger.
			WithFields(logger.Fields{
				"key":        key.String(""),
				"reason":     "summarize simple route failed",
				"error":      err,
				"request_id": requestid.RequestIDFromCtx(ctx),
			}).
			Info("cache missed")
		metrics.IncrFindRouteCacheCount(false, []string{"reason:summarizeCachedRouteFailed"})

		return nil, err
	}

	priceImpact := routeSummary.GetPriceImpact()

	if priceImpact > c.config.PriceImpactThreshold {
		logger.
			WithFields(logger.Fields{
				"key":        key.String(""),
				"reason":     "price impact is greater than threshold",
				"error":      err,
				"request_id": requestid.RequestIDFromCtx(ctx),
			}).
			Info("cache missed")
		metrics.IncrFindRouteCacheCount(
			false,
			[]string{
				"reason:priceImpactIsGreaterThanEpsilon",
				fmt.Sprintf("priceImpact:%f", priceImpact),
			},
		)

		return nil, errors.Wrapf(
			ErrPriceImpactIsGreaterThanThreshold,
			"priceImpact: [%f]",
			priceImpact,
		)
	}

	logger.
		WithFields(logger.Fields{
			"key":        key.String(""),
			"request_id": requestid.RequestIDFromCtx(ctx),
		}).
		Info("cache hit")
	metrics.IncrFindRouteCacheCount(true, nil)

	return routeSummary, nil
}

func (c *cache) setRouteToCache(ctx context.Context, routeSummary *valueobject.RouteSummary, key *valueobject.RouteCacheKey, ttl time.Duration) {
	simpleRoute := simplifyRouteSummary(routeSummary)

	if err := c.routeCacheRepository.Set(ctx, key, simpleRoute, ttl); err != nil {
		logger.
			WithFields(logger.Fields{"error": err}).
			Error("cache.setRouteToCache failed")
	}
}

func (c *cache) getCachePointTTL(amount float64) (time.Duration, bool) {
	for _, cachePoint := range c.config.TTLByAmount {
		if float64AlmostEqual(cachePoint.Amount, amount) {
			return cachePoint.TTL, true
		}
	}

	return 0, false
}

func (c *cache) genKey(params *types.AggregateParams) (*valueobject.RouteCacheKey, time.Duration) {
	amountInWithoutDecimals := business.AmountWithoutDecimals(params.AmountIn, params.TokenIn.Decimals)
	amountInWithoutDecimalsFloat64, _ := amountInWithoutDecimals.Float64()

	ttlByAmount, ok := c.getCachePointTTL(amountInWithoutDecimalsFloat64)
	if ok {
		return &valueobject.RouteCacheKey{
			CacheMode:  valueobject.RouteCacheModePoint,
			TokenIn:    params.TokenIn.Address,
			TokenOut:   params.TokenOut.Address,
			AmountIn:   strconv.FormatFloat(amountInWithoutDecimalsFloat64, 'f', -1, 64),
			SaveGas:    params.SaveGas,
			GasInclude: params.GasInclude,
			Dexes:      params.Sources,
		}, ttlByAmount
	}

	amountInUSD := business.CalcAmountUSD(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD)
	amountInUSDFloat64, _ := amountInUSD.Float64()

	shrunkAmountInUSD := c.shrinkFunc(amountInUSDFloat64)

	for _, cacheRange := range c.config.TTLByAmountUSDRange {
		if shrunkAmountInUSD > cacheRange.AmountUSDLowerBound {
			return &valueobject.RouteCacheKey{
				CacheMode:  valueobject.RouteCacheModeRange,
				TokenIn:    params.TokenIn.Address,
				TokenOut:   params.TokenOut.Address,
				AmountIn:   strconv.FormatFloat(shrunkAmountInUSD, 'f', -1, 64),
				SaveGas:    params.SaveGas,
				GasInclude: params.GasInclude,
				Dexes:      params.Sources,
			}, cacheRange.TTL
		}
	}

	return &valueobject.RouteCacheKey{
		CacheMode:  valueobject.RouteCacheModeRange,
		TokenIn:    params.TokenIn.Address,
		TokenOut:   params.TokenOut.Address,
		AmountIn:   strconv.FormatFloat(shrunkAmountInUSD, 'f', -1, 64),
		SaveGas:    params.SaveGas,
		GasInclude: params.GasInclude,
		Dexes:      params.Sources,
	}, c.config.DefaultTTL
}

func (c *cache) summarizeSimpleRoute(
	ctx context.Context,
	simpleRoute *valueobject.SimpleRoute,
	params *types.AggregateParams,
) (*valueobject.RouteSummary, error) {
	// Step 1: prepare pool data
	poolByAddress, err := c.poolManager.GetPoolByAddress(
		ctx,
		simpleRoute.ExtractPoolAddresses(),
		params.Sources,
	)
	if err != nil {
		return nil, err
	}

	poolBucket := valueobject.NewPoolBucket(poolByAddress)

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
				return nil, errors.Wrapf(
					ErrInvalidSwap,
					"cache.summarizeSimpleRoute > pool not found [%s]",
					simpleSwap.PoolAddress,
				)
			}

			// Step 3.1.2: simulate c swap through the pool
			result, err := pool.CalcAmountOut(tokenAmountIn, simpleSwap.TokenOutAddress)
			if err != nil {
				return nil, errors.Wrapf(
					ErrInvalidSwap,
					"cache.summarizeSimpleRoute > swap failed > pool: [%s] > error : [%v]",
					simpleSwap.PoolAddress,
					err,
				)
			}

			// Step 3.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.Wrapf(
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
			}
			pool = poolBucket.ClonePool(simpleSwap.PoolAddress)
			pool.UpdateBalance(updateBalanceParams)

			// Step 3.1.5: summarize the swap
			swap := valueobject.Swap{
				Pool:              simpleSwap.PoolAddress,
				TokenIn:           simpleSwap.TokenInAddress,
				TokenOut:          simpleSwap.TokenOutAddress,
				SwapAmount:        tokenAmountIn.Amount,
				AmountOut:         result.TokenAmountOut.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(pool.GetExchange()),
				PoolLength:        len(pool.GetTokens()),
				PoolType:          pool.GetType(),
				PoolExtra:         pool.GetMetaInfo(simpleSwap.TokenInAddress, simpleSwap.TokenOutAddress),
				Extra:             result.SwapInfo,
			}

			summarizedPath = append(summarizedPath, swap)

			// Step 3.1.6: add up gas fee
			gas += result.Gas

			// Step 3.1.7: update input of the next swap is output of current swap
			tokenAmountIn = *result.TokenAmountOut
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
