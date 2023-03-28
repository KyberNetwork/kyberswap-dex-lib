package repository

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type PriceRepository struct {
	IPriceDatastoreRepository
	IPriceCacheRepository
}

func NewPriceRepository(
	datastoreRepo IPriceDatastoreRepository,
	cacheRepo IPriceCacheRepository,
) *PriceRepository {
	return &PriceRepository{
		IPriceDatastoreRepository: datastoreRepo,
		IPriceCacheRepository:     cacheRepo,
	}
}

func (r *PriceRepository) Save(ctx context.Context, price entity.Price) error {
	if err := r.Persist(ctx, price); err != nil {
		return err
	}

	if err := r.Set(ctx, price.Address, price); err != nil {
		return err
	}

	return nil
}
