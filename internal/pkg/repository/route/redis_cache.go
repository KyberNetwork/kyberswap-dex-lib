package route

import (
	"context"
	"strconv"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
)

type redisCacheRepository struct {
	cache *cache.Cache

	config RedisCacheRepositoryConfig
}

func NewRedisCacheRepository(redisClient redis.UniversalClient, config RedisCacheRepositoryConfig) *redisCacheRepository {
	cacheOptions := &cache.Options{
		Redis:      redisClient,
		LocalCache: cache.NewTinyLFU(config.LocalCacheSize, config.LocalCacheTTL),
	}

	return &redisCacheRepository{
		cache:  cache.New(cacheOptions),
		config: config,
	}
}

// Set saves route to cache
func (r *redisCacheRepository) Set(
	ctx context.Context,
	key *valueobject.RouteCacheKey,
	route *valueobject.SimpleRoute,
	ttl time.Duration,
) error {
	item := &cache.Item{
		Ctx:   ctx,
		Key:   r.genKey(key),
		Value: route,
		TTL:   ttl,
	}

	return r.cache.Set(item)
}

// Get returns route from cache if it's exists and valid
func (r *redisCacheRepository) Get(
	ctx context.Context,
	key *valueobject.RouteCacheKey,
) (*valueobject.SimpleRoute, error) {
	var simpleRoute valueobject.SimpleRoute
	if err := r.cache.Get(ctx, r.genKey(key), &simpleRoute); err != nil {
		return nil, err
	}

	return &simpleRoute, nil
}

func (r *redisCacheRepository) genKey(key *valueobject.RouteCacheKey) string {
	return strconv.FormatUint(key.Hash(r.config.Prefix), 10)
}
