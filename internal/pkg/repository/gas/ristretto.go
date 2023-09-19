package gas

import (
	"context"
	"math/big"

	"github.com/dgraph-io/ristretto"
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

func (r *ristrettoRepository) GetSuggestedGasPrice(ctx context.Context) (*big.Int, error) {
	cachedSuggestedGasPrice, found := r.cache.Get(CacheKeySuggestedGasPrice)
	if found {
		suggestedGasPrice, ok := cachedSuggestedGasPrice.(*big.Int)
		if ok {
			return suggestedGasPrice, nil
		}
	}

	suggestedGasPrice, err := r.fallbackRepository.GetSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	r.cache.SetWithTTL(CacheKeySuggestedGasPrice, suggestedGasPrice, r.config.SuggestedGasPrice.Cost, r.config.SuggestedGasPrice.TTL)

	return suggestedGasPrice, nil
}
