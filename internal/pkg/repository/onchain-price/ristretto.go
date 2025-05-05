package onchainprice

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/backoff"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type ristrettoRepository struct {
	cache          *ristretto.Cache
	grpcRepository *grpcRepository
	config         RistrettoConfig
	nativeUSDPrice atomic.Pointer[big.Float]
}

const (
	CacheKeyNativeUsd = "native-token-usd-price"
)

var (
	zeroBF = big.NewFloat(0)

	ErrNativeUSDPriceNotFoundInCache = errors.New("native usd price not found in cache")
)

func NewRistrettoRepository(
	grpcRepository *grpcRepository,
	config RistrettoConfig,
) (*ristrettoRepository, error) {

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
		Metrics:     true,
	})
	if err != nil {
		return nil, err
	}

	r := &ristrettoRepository{
		cache:          cache,
		grpcRepository: grpcRepository,
		config:         config,
	}

	r.nativeUSDPrice.Store(zeroBF)

	return r, nil
}

func (r *ristrettoRepository) FindByAddresses(ctx context.Context, addresses []string) (map[string]*entity.OnchainPrice, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[onchainprice] ristrettoRepository.FindByAddresses")
	defer span.End()

	prices := make(map[string]*entity.OnchainPrice, len(addresses))
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		cachedPrice, found := r.cache.Get(address)
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		price, ok := cachedPrice.(*entity.OnchainPrice)
		if !ok {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		prices[address] = price
	}
	if len(prices) != 0 {
		metrics.CountPriceHitLocalCache(ctx, int64(len(prices)), true)
	}

	if len(uncachedAddresses) == 0 {
		return prices, nil
	}
	metrics.CountPriceHitLocalCache(ctx, int64(len(uncachedAddresses)), false)

	uncachedPrices, err := r.grpcRepository.FindByAddresses(ctx, uncachedAddresses)
	if err != nil {
		logger.Errorf(ctx, "[onchainprice] ristrettoRepository.FindByAddresses GetUncachedPrices err: %v", err)
		// just return what we have instead of discarding everything
		return prices, nil
	}

	nativePriceInUsd, err := r.GetNativePriceInUsd(ctx)
	if err != nil {
		logger.Errorf(ctx, "[onchainprice] ristrettoRepository.FindByAddresses GetNativePriceInUsd %v", err)
		return prices, nil
	}

	for address, price := range uncachedPrices {
		prices[address] = price
		if price.NativePrice.Buy != nil {
			price.USDPrice.Buy = new(big.Float).Mul(price.NativePrice.Buy, nativePriceInUsd)
		}
		if price.NativePrice.Sell != nil {
			price.USDPrice.Sell = new(big.Float).Mul(price.NativePrice.Sell, nativePriceInUsd)
		}
	}

	for address, price := range uncachedPrices {
		r.cache.SetWithTTL(address, price, r.config.Price.Cost, r.config.Price.TTL)
	}

	return prices, nil
}

func (r *ristrettoRepository) GetNativePriceInUsd(ctx context.Context) (*big.Float, error) {
	if nativeUSDPrice := r.nativeUSDPrice.Load(); nativeUSDPrice.Sign() > 0 {
		return nativeUSDPrice, nil
	}

	return zeroBF, ErrNativeUSDPriceNotFoundInCache
}

func (r *ristrettoRepository) RefreshCacheNativePriceInUSD(ctx context.Context) {
	// fetch native price in usd every half of TTL to make sure that we always have the latest price available from cache
	ticker := time.NewTicker(r.config.Price.TTL / 2)
	defer ticker.Stop()

	for {
		_ = backoff.RetryE(func() error {
			if err := r.fetchNativePriceInUSD(ctx); err != nil {
				logger.Errorf(ctx, "failed to fetch native price in usd: %v", err)
				return err
			}

			return nil
		})

		select {
		case <-ctx.Done():
			logger.Infof(ctx, "stop fetching native price in usd with error: %v", ctx.Err())
			return
		case <-ticker.C:
			continue
		}
	}
}

func (r *ristrettoRepository) fetchNativePriceInUSD(ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "[onchainprice] ristrettoRepository.fetchNativePriceInUSD")
	defer span.End()

	price, err := r.grpcRepository.GetNativePriceInUsd(ctx)
	if err != nil {
		return err
	}

	if price == nil || price.Sign() <= 0 {
		return err
	}

	// Set native price in usd to the atomic pointer
	r.nativeUSDPrice.Store(price)

	logger.Debugf(ctx, "refresh cache with native price in usd: %s", price.String())

	return nil
}
