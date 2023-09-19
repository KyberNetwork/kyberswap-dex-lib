package price

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/dgraph-io/ristretto"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type ristrettoRepository struct {
	cache              *ristretto.Cache
	fallbackRepository IFallbackRepository
	config             RistrettoConfig
}

func NewRistrettoRepository(
	fallbackRepository IFallbackRepository,
	config RistrettoConfig,
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
		cache:              cache,
		fallbackRepository: fallbackRepository,
		config:             config,
	}, nil
}

func (r *ristrettoRepository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[price] ristrettoRepository.FindByAddresses")
	defer span.Finish()

	prices := make([]*entity.Price, 0, len(addresses))
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		cachedPrice, found := r.cache.Get(address)
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		price, ok := cachedPrice.(*entity.Price)
		if !ok {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		prices = append(prices, price)
	}

	if len(uncachedAddresses) == 0 {
		return prices, nil
	}

	uncachedPrices, err := r.fallbackRepository.FindByAddresses(ctx, uncachedAddresses)
	if err != nil {
		return nil, err
	}

	prices = append(prices, uncachedPrices...)

	for _, token := range uncachedPrices {
		r.cache.SetWithTTL(token.Address, token, r.config.Price.Cost, r.config.Price.TTL)
	}

	return prices, nil
}
