package usecase

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type CacheRouteUseCase struct {
	config         CacheRouteConfig
	routeCacheRepo IRouteCacheRepository
}

func NewCacheRouteUseCase(
	config CacheRouteConfig,
	routeCacheRepo IRouteCacheRepository,
) *CacheRouteUseCase {
	return &CacheRouteUseCase{
		config:         config,
		routeCacheRepo: routeCacheRepo,
	}
}

func (u *CacheRouteUseCase) Get(
	ctx context.Context,
	key valueobject.RouteCacheKey,
) (*core.CachedRoute, error) {
	data, ttl, err := u.routeCacheRepo.Get(ctx, key.String(u.config.KeyPrefix))
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, errors.Wrapf(
			ErrRouteCacheNotFound,
			"[CacheRouteUseCase.Get] key: [%s]",
			key.String(u.config.KeyPrefix),
		)
	}

	if ttl <= 0 {
		return nil, errors.Wrapf(
			ErrRouteCacheExpired,
			"[CacheRouteUseCase.Get] key: [%s]",
			key.String(u.config.KeyPrefix),
		)
	}

	var cachedRoute core.CachedRoute
	if err = json.Unmarshal(data, &cachedRoute); err != nil {
		return nil, errors.Wrapf(
			ErrRouteCacheUnmarshalFailed,
			"[CacheRouteUseCase.Get] key: [%s], err: [%v]",
			key.String(u.config.KeyPrefix),
			err,
		)
	}

	return &cachedRoute, nil
}

// Set stores route to cache
func (u *CacheRouteUseCase) Set(
	ctx context.Context,
	key valueobject.RouteCacheKey,
	route core.CachedRoute,
	amountIn *big.Int,
	decimals uint8,
	amountInUSD int,
) error {
	amount := new(big.Float).Quo(
		new(big.Float).SetInt(amountIn),
		constant.TenPowDecimals(decimals),
	)

	ttl := u.GetCacheTTL(amount, amountInUSD)

	data, err := json.Marshal(route)
	if err != nil {
		return err
	}

	return u.routeCacheRepo.Set(ctx, key.String(u.config.KeyPrefix), string(data), ttl)
}

// GenKey generates a RouteCacheKey based on input
func (u *CacheRouteUseCase) GenKey(
	tokenIn string,
	tokenOut string,
	amountIn *big.Int,
	decimals uint8,
	amountInUSD int,
	saveGas bool,
	dexes []string,
	gasInclude bool,

) valueobject.RouteCacheKey {
	amount := new(big.Float).Quo(
		new(big.Float).SetInt(amountIn),
		constant.TenPowDecimals(decimals),
	)

	if u.isCachePoint(amount) {
		return valueobject.RouteCacheKey{
			TokenIn:    tokenIn,
			TokenOut:   tokenOut,
			SaveGas:    saveGas,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   amountIn.String(),
			Dexes:      dexes,
			GasInclude: gasInclude,
		}
	}

	return valueobject.RouteCacheKey{
		TokenIn:    tokenIn,
		TokenOut:   tokenOut,
		SaveGas:    saveGas,
		CacheMode:  valueobject.RouteCacheModeRange,
		AmountIn:   strconv.Itoa(amountInUSD),
		Dexes:      dexes,
		GasInclude: gasInclude,
	}
}

// GetCacheTTL receives amount and amountUSD and returns ttl of the cache
// - amount is amountIn divided by decimals
// - amountUSD is amountIn in USD
func (u *CacheRouteUseCase) GetCacheTTL(amount *big.Float, amountUSD int) time.Duration {
	for _, cachePoint := range u.config.CachePoints {
		cachePointAmount := new(big.Float).SetInt64(cachePoint.Amount)

		if amount.Cmp(cachePointAmount) == 0 {
			return cachePoint.TTL
		}
	}

	for _, cacheRange := range u.config.CacheRanges {
		if cacheRange.FromUSD <= amountUSD && amountUSD <= cacheRange.ToUSD {
			return cacheRange.TTL
		}
	}

	return u.config.DefaultCacheTTL
}

// isCachePoint receives amount and returns true if amount matched with a cached points
// - amount is amountIn divided by decimals
func (u *CacheRouteUseCase) isCachePoint(amount *big.Float) bool {
	for _, cachePoint := range u.config.CachePoints {
		cachePointAmount := new(big.Float).SetInt64(cachePoint.Amount)

		if amount.Cmp(cachePointAmount) == 0 {
			return true
		}
	}

	return false
}
