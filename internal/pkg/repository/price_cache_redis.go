package repository

import (
	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"
)

type PriceCacheRedisRepository struct {
	cache *redis.Redis
}

func NewPriceCacheRedisRepository(cache *redis.Redis) *PriceCacheRedisRepository {
	return &PriceCacheRedisRepository{
		cache: cache,
	}
}

func (r *PriceCacheRedisRepository) Keys(ctx context.Context) []string {
	keys, err := r.cache.Client.HKeys(
		ctx,
		r.cache.FormatKey(entity.PriceKey),
	).Result()

	if err != nil {
		return []string{}
	}

	return keys
}

func (r *PriceCacheRedisRepository) Get(ctx context.Context, address string) (entity.Price, error) {
	priceString, err := r.cache.Client.HGet(
		ctx,
		r.cache.FormatKey(entity.PriceKey),
		address,
	).Result()
	if err != nil {
		return entity.Price{
			Address: address,
		}, ErrPriceNotFoundInCache
	}

	return entity.DecodePrice(address, priceString), nil
}

func (r *PriceCacheRedisRepository) Set(ctx context.Context, address string, price entity.Price) error {
	return r.cache.Client.HSet(
		ctx,
		r.cache.FormatKey(entity.PriceKey),
		address,
		price.Encode(),
	).Err()
}

func (r *PriceCacheRedisRepository) Remove(ctx context.Context, address string) error {
	return r.cache.Client.HDel(
		ctx,
		r.cache.FormatKey(entity.PriceKey),
		address,
	).Err()
}

func (r *PriceCacheRedisRepository) Count(ctx context.Context) int {
	numPrices, err := r.cache.Client.HLen(
		ctx,
		r.cache.FormatKey(entity.PriceKey),
	).Result()

	if err != nil {
		return 0
	}

	return int(numPrices)
}
