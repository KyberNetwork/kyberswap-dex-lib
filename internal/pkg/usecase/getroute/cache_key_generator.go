package getroute

import (
	"context"
	"errors"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
)

type routeKeyGenerator struct {
	mu     sync.RWMutex
	config valueobject.CacheConfig

	amountInUsdShrinkFunc  ShrinkFunc
	amountInShrinkFuncList []ShrinkFunc
}

func newCacheKeyGenerator(
	config valueobject.CacheConfig,
) *routeKeyGenerator {
	amountInUsdShrinkFunc, _ := ShrinkFuncFactory(ShrinkFuncName(config.ShrinkFuncName), map[ShrinkFuncConfig]float64{
		ShrinkFuncPowExp:     config.ShrinkFuncPowExp,
		ShrinkDecimalBase:    config.ShrinkDecimalBase,
		ShrinkFuncLogPercent: config.ShrinkFuncLogPercent,
	})
	amountInShrinkFuncList := []ShrinkFunc{}
	for _, c := range config.ShrinkAmountInConfigs {
		amountInShrinkFunc, _ := ShrinkFuncFactory(ShrinkFuncName(c.ShrinkFuncName), map[ShrinkFuncConfig]float64{
			ShrinkDecimalBase:    c.ShrinkFuncConstant,
			ShrinkFuncLogPercent: c.ShrinkFuncConstant,
		})
		amountInShrinkFuncList = append(amountInShrinkFuncList, amountInShrinkFunc)
	}

	return &routeKeyGenerator{
		config:                 config,
		amountInUsdShrinkFunc:  amountInUsdShrinkFunc,
		amountInShrinkFuncList: amountInShrinkFuncList,
	}
}

func (g *routeKeyGenerator) genKey(ctx context.Context, params *types.AggregateParams) (mapset.Set[valueobject.RouteCacheKeyTTL], error) {
	// If request has excluded more than 1 pool, we will not hit cache.
	if params.ExcludedPools != nil && params.ExcludedPools.Cardinality() > 1 {
		metrics.IncrFindRouteCacheCount(ctx, false, map[string]string{
			"excludePools": "true",
		})
		return mapset.NewSet[valueobject.RouteCacheKeyTTL](), nil
	}

	if g.config.EnableNewCacheKeyGenerator {
		return g.genKeyV2(params)
	}

	return g.genKeyV1(params)
}

// genKey retrieves the key required to access the cacheRoute.
// It returns an error if these parameters do not correspond to a cache point and lack pricing information.
func (g *routeKeyGenerator) genKeyV1(params *types.AggregateParams) (mapset.Set[valueobject.RouteCacheKeyTTL], error) {
	key, duration, _ := g.genKeyByCachePointTTL(params)
	// cache point ttl has been found in the config
	if key != nil {
		return mapset.NewSet[valueobject.RouteCacheKeyTTL](valueobject.RouteCacheKeyTTL{Key: key, TTL: duration}), nil
	}

	// if this token in doesn't have price (considered token has no price if tokenInUSD is too small), return error
	if params.TokenInPriceUSD <= g.config.MinAmountInUSD {
		return mapset.NewSet[valueobject.RouteCacheKeyTTL](), ErrNoTokenInPrice
	}

	return g.genKeyByAmountInUSD(params)
}

func (g *routeKeyGenerator) genKeyV2(params *types.AggregateParams) (mapset.Set[valueobject.RouteCacheKeyTTL], error) {
	// if this token in doesn't have price, cache route by AmountIn, otherwise cache them by AmountInUSD
	if params.TokenInPriceUSD <= g.config.MinAmountInUSD {
		return g.genKeyByAmountIn(params)
	}

	return g.genKeyByAmountInUSD(params)
}

func (g *routeKeyGenerator) genKeyByCachePointTTL(params *types.AggregateParams) (*valueobject.RouteCacheKey, time.Duration, error) {
	amountInWithoutDecimals := business.AmountWithoutDecimals(params.AmountIn, params.TokenIn.Decimals)
	amountInWithoutDecimalsFloat64, _ := amountInWithoutDecimals.Float64()

	for _, cachePoint := range g.config.TTLByAmount {
		if utils.Float64AlmostEqual(cachePoint.Amount, amountInWithoutDecimalsFloat64) {
			return &valueobject.RouteCacheKey{
				CacheMode:              valueobject.RouteCacheModePoint,
				TokenIn:                params.TokenIn.Address,
				TokenOut:               params.TokenOut.Address,
				AmountIn:               strconv.FormatFloat(amountInWithoutDecimalsFloat64, 'f', -1, 64),
				SaveGas:                params.SaveGas,
				GasInclude:             params.GasInclude,
				Dexes:                  params.Sources,
				IsPathGeneratorEnabled: params.IsPathGeneratorEnabled,
				IsHillClimbingEnabled:  params.IsHillClimbEnabled,
				ExcludedPools:          setToSlice(params.ExcludedPools),
			}, cachePoint.TTL, nil
		}
	}

	return nil, 0, errors.New("cache point not found in config")
}

func (g *routeKeyGenerator) genKeyByAmountInUSD(params *types.AggregateParams) (mapset.Set[valueobject.RouteCacheKeyTTL], error) {
	if params.TokenIn.Decimals <= 0 {
		return mapset.NewSet[valueobject.RouteCacheKeyTTL](), errors.New("token decimal has not been found")
	}
	amountInUSD := business.CalcAmountUSD(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD)
	amountInUSDFloat64, _ := amountInUSD.Float64()

	shrunkAmountInUSD := g.amountInUsdShrinkFunc(amountInUSDFloat64)
	ttl := g.config.DefaultTTL

	for _, cacheRange := range g.config.TTLByAmountUSDRange {
		if shrunkAmountInUSD > cacheRange.AmountUSDLowerBound {
			ttl = cacheRange.TTL
		}
	}

	return mapset.NewSet[valueobject.RouteCacheKeyTTL](valueobject.RouteCacheKeyTTL{
		Key: &valueobject.RouteCacheKey{
			CacheMode:              valueobject.RouteCacheModeRangeByUSD,
			TokenIn:                params.TokenIn.Address,
			TokenOut:               params.TokenOut.Address,
			AmountIn:               strconv.FormatFloat(shrunkAmountInUSD, 'f', -1, 64),
			SaveGas:                params.SaveGas,
			GasInclude:             params.GasInclude,
			Dexes:                  params.Sources,
			IsPathGeneratorEnabled: params.IsPathGeneratorEnabled,
			IsHillClimbingEnabled:  params.IsHillClimbEnabled,
			ExcludedPools:          setToSlice(params.ExcludedPools),
		},
		TTL: ttl,
	}), nil
}

func (g *routeKeyGenerator) genKeyByAmountIn(params *types.AggregateParams) (mapset.Set[valueobject.RouteCacheKeyTTL], error) {
	if params.TokenIn.Decimals <= 0 {
		return mapset.NewSet[valueobject.RouteCacheKeyTTL](), errors.New("token decimal has not been found")
	}
	ttl := g.config.DefaultTTL
	for _, cacheRange := range g.config.TTLByAmountRange {
		if params.AmountIn.Cmp(cacheRange.AmountLowerBound) >= 0 {
			ttl = cacheRange.TTL
		}
	}

	amountInWithoutDecimals := business.AmountWithoutDecimals(params.AmountIn, params.TokenIn.Decimals)
	amountInWithoutDecimalsFloat64, _ := amountInWithoutDecimals.Float64()
	shrunkAmountInSet := mapset.NewSet[valueobject.RouteCacheKeyTTL]()
	seenAmount := mapset.NewSet[string]()

	for _, f := range g.amountInShrinkFuncList {
		shrunkAmountIn := f(amountInWithoutDecimalsFloat64)
		// We don't get route from cache if the amountIn is too large, because it can cause a large slippage, ignore the large point
		if math.Abs(shrunkAmountIn-amountInWithoutDecimalsFloat64) >= g.config.ShrinkAmountInThreshold {
			logger.Warnf("different between shunk value and amount in without decimal is above threshold")
		} else {
			amount := strconv.FormatFloat(shrunkAmountIn, 'f', 0, 64)
			if seenAmount.Contains(amount) {
				continue
			}
			seenAmount.Add(amount)
			shrunkAmountInSet.Add(valueobject.RouteCacheKeyTTL{
				Key: &valueobject.RouteCacheKey{
					CacheMode:              valueobject.RouteCacheModeRangeByAmount,
					TokenIn:                params.TokenIn.Address,
					TokenOut:               params.TokenOut.Address,
					AmountIn:               amount,
					SaveGas:                params.SaveGas,
					GasInclude:             params.GasInclude,
					Dexes:                  params.Sources,
					IsPathGeneratorEnabled: params.IsPathGeneratorEnabled,
					IsHillClimbingEnabled:  params.IsHillClimbEnabled,
					ExcludedPools:          setToSlice(params.ExcludedPools),
				},
				TTL: ttl,
			})
		}
	}

	if shrunkAmountInSet.IsEmpty() {
		return mapset.NewSet[valueobject.RouteCacheKeyTTL](), errors.New("different between shunk value and amount in without decimal is above threshold")
	}

	return shrunkAmountInSet, nil
}

func (g *routeKeyGenerator) applyConfig(config Config) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// only apply cache only if it changed
	if !g.config.Equals(config.Cache) {

		if amountInUsdShrinkFunc, err := ShrinkFuncFactory(ShrinkFuncName(config.Cache.ShrinkFuncName), map[ShrinkFuncConfig]float64{
			ShrinkFuncPowExp:     config.Cache.ShrinkFuncPowExp,
			ShrinkDecimalBase:    config.Cache.ShrinkDecimalBase,
			ShrinkFuncLogPercent: config.Cache.ShrinkFuncLogPercent,
		}); err != nil {
			logger.Errorf("Can not apply amountInUsdShrinkFunc from remote config err %e", err)
		} else {
			g.amountInUsdShrinkFunc = amountInUsdShrinkFunc
		}
		amountInShrinkFuncList := []ShrinkFunc{}
		for _, c := range config.Cache.ShrinkAmountInConfigs {
			if amountInShrinkFunc, err := ShrinkFuncFactory(ShrinkFuncName(c.ShrinkFuncName), map[ShrinkFuncConfig]float64{
				ShrinkDecimalBase:    c.ShrinkFuncConstant,
				ShrinkFuncLogPercent: c.ShrinkFuncConstant,
			}); err != nil {
				logger.Errorf("Can not apply amountInShrinkFunc from remote config err %e", err)
			} else {
				amountInShrinkFuncList = append(amountInShrinkFuncList, amountInShrinkFunc)
			}
		}
		if len(amountInShrinkFuncList) == len(g.amountInShrinkFuncList) {
			g.amountInShrinkFuncList = amountInShrinkFuncList
		}
		g.config.EnableNewCacheKeyGenerator = config.Cache.EnableNewCacheKeyGenerator
		g.config.ShrinkAmountInThreshold = config.Cache.ShrinkAmountInThreshold
	}
	g.config = config.Cache
}
