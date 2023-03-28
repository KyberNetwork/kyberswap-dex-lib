package repository

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

type PoolDatastoreRedisRepository struct {
	db *redis.Redis
}

func NewPoolDataStoreRedisRepository(
	db *redis.Redis,
) *PoolDatastoreRedisRepository {
	return &PoolDatastoreRedisRepository{
		db: db,
	}
}

func (r *PoolDatastoreRedisRepository) FindAll(
	ctx context.Context,
) ([]entity.Pool, error) {
	poolMap, err := r.db.Client.HGetAll(
		ctx,
		r.db.FormatKey(entity.PoolKey),
	).Result()

	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolMap))

	for key, poolString := range poolMap {
		pool, derr := entity.DecodePool(key, poolString)
		if derr != nil {
			return nil, derr
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

func (r *PoolDatastoreRedisRepository) FindByAddresses(
	ctx context.Context,
	addresses []string,
) ([]entity.Pool, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	poolStrings, err := r.db.Client.HMGet(
		ctx,
		r.db.FormatKey(entity.PoolKey),
		addresses...,
	).Result()

	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolStrings))

	for i, poolString := range poolStrings {
		if poolString != nil {
			decodedPool, err := entity.DecodePool(addresses[i], poolString.(string))
			if err != nil {
				return nil, err
			}

			pools = append(pools, decodedPool)
		}
	}

	return pools, nil
}

func (r *PoolDatastoreRedisRepository) Persist(
	ctx context.Context,
	pool entity.Pool,
) error {
	encodedPool, err := pool.Encode()

	if err != nil {
		return err
	}

	_, err = r.db.Client.HSet(
		ctx,
		r.db.FormatKey(entity.PoolKey),
		pool.Address,
		encodedPool,
	).Result()

	return err
}

func (r *PoolDatastoreRedisRepository) Delete(
	ctx context.Context,
	pool entity.Pool,
) error {
	_, err := r.db.Client.HDel(
		ctx,
		r.db.FormatKey(entity.PoolKey),
		pool.Address,
	).Result()

	return err
}
