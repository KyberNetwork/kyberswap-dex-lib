package onchainprice

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
	"github.com/dgraph-io/ristretto"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type ristrettoRepository struct {
	cache          *ristretto.Cache
	grpcRepository *grpcRepository
	config         price.RistrettoConfig
}

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
	defer span.Finish()

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
		return nil, err
	}

	for address, price := range uncachedPrices {
		prices[address] = price
	}

	for address, price := range uncachedPrices {
		r.cache.SetWithTTL(address, price, r.config.Price.Cost, r.config.Price.TTL)
	}

	return prices, nil
}
