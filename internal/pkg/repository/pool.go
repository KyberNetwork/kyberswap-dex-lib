package repository

import (
	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
)

type PoolRepository struct {
	IPoolDatastoreRepository
	IPoolCacheRepository
}

func NewPoolRepository(
	datastoreRepo IPoolDatastoreRepository,
	cacheRepo IPoolCacheRepository,
) *PoolRepository {
	return &PoolRepository{
		IPoolDatastoreRepository: datastoreRepo,
		IPoolCacheRepository:     cacheRepo,
	}
}

func (r *PoolRepository) Save(ctx context.Context, pool entity.Pool) error {
	// Save pool into database
	if err := r.Persist(ctx, pool); err != nil {
		return err
	}

	// Save pool into cache
	if err := r.Set(ctx, pool.Address, pool); err != nil {
		return err
	}

	return nil
}
