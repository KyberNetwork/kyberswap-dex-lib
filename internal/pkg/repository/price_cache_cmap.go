package repository

import (
	"context"
	"errors"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"

	cmap "github.com/orcaman/concurrent-map"
)

var ErrPriceNotFoundInCache = errors.New("can not get price in cache")

type PriceCacheCMapRepository struct {
	priceMap cmap.ConcurrentMap
}

func NewPriceCacheCMapRepository(cache cmap.ConcurrentMap) *PriceCacheCMapRepository {
	return &PriceCacheCMapRepository{
		priceMap: cache,
	}
}

func (r *PriceCacheCMapRepository) Keys(_ context.Context) []string {
	return r.priceMap.Keys()
}

func (r *PriceCacheCMapRepository) Get(_ context.Context, address string) (entity.Price, error) {
	price, ok := r.priceMap.Get(address)
	if !ok {
		return entity.Price{
			Address: address,
		}, ErrPriceNotFoundInCache
	}

	return price.(entity.Price), nil
}

func (r *PriceCacheCMapRepository) Set(_ context.Context, address string, price entity.Price) error {
	r.priceMap.Set(address, price)
	return nil
}

func (r *PriceCacheCMapRepository) Remove(_ context.Context, address string) error {
	r.priceMap.Remove(address)
	return nil
}

func (r *PriceCacheCMapRepository) Count(_ context.Context) int {
	return r.priceMap.Count()
}
