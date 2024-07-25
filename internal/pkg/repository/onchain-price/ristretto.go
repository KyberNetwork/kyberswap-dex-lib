package onchainprice

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/dgraph-io/ristretto"
)

type ristrettoRepository struct {
	cache          *ristretto.Cache
	grpcRepository *grpcRepository
	config         price.RistrettoConfig
}

const (
	CacheKeyNativeUsd = "native-token-usd-price"
)

func NewRistrettoRepository(
	grpcRepository *grpcRepository,
	config price.RistrettoConfig,
) (*ristrettoRepository, error) {

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
	})
	if err != nil {
		return nil, err
	}

	return &ristrettoRepository{
		cache:          cache,
		grpcRepository: grpcRepository,
		config:         config,
	}, nil
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

	if len(uncachedAddresses) == 0 {
		return prices, nil
	}

	uncachedPrices, err := r.grpcRepository.FindByAddresses(ctx, uncachedAddresses)
	if err != nil {
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
	span, ctx := tracer.StartSpanFromContext(ctx, "[onchainprice] ristrettoRepository.GetNativePriceInUsd")
	defer span.End()

	if cachedPrice, found := r.cache.Get(CacheKeyNativeUsd); found {
		if price, ok := cachedPrice.(*big.Float); ok {
			return price, nil
		}
	}

	price, err := r.grpcRepository.GetNativePriceInUsd(ctx)
	if err != nil {
		return nil, err
	}

	r.cache.SetWithTTL(CacheKeyNativeUsd, price, r.config.Price.Cost, r.config.Price.TTL)
	return price, nil
}
