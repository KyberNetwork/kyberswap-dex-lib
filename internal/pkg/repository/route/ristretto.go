package route

import (
	"context"
	"strconv"

	"github.com/dgraph-io/ristretto"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func genKey(key valueobject.RouteCacheKeyTTL, prefix string) string {
	return utils.Join(prefix, strconv.FormatUint(key.Key.Hash(prefix), 10))
}

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

func (r *ristrettoRepository) Get(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) (map[valueobject.RouteCacheKeyTTL]*valueobject.SimpleRouteWithExtraData, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[route] ristrettoRepository.Get")
	defer span.End()

	routes := map[valueobject.RouteCacheKeyTTL]*valueobject.SimpleRouteWithExtraData{}
	uncachedKeys := make([]valueobject.RouteCacheKeyTTL, 0, len(keys))

	for _, key := range keys {
		cacheKey := genKey(key, r.config.Prefix)
		cachedRoute, found := r.cache.Get(cacheKey)
		if !found {
			uncachedKeys = append(uncachedKeys, key)
			continue
		}

		route, ok := cachedRoute.(*valueobject.SimpleRouteWithExtraData)
		if !ok {
			uncachedKeys = append(uncachedKeys, key)
			continue
		}

		log.Ctx(ctx).Debug().Str("key", cacheKey).Msg("[route] istrettoRepository.Get hit local cache")
		routes[key] = route
	}

	if len(uncachedKeys) == 0 {
		return routes, nil
	}

	uncachedRoutes, err := r.fallbackRepository.Get(ctx, uncachedKeys)
	if err != nil {
		return nil, err
	}

	// When we set a route to local cache after we get it from redis, we have to accept min TTL in the config
	// because we don't know how long it has been retained in Redis
	for key, route := range uncachedRoutes {
		if route == nil {
			continue
		}
		cacheKey := genKey(key, r.config.Prefix)
		r.cache.SetWithTTL(cacheKey, route, r.config.Route.Cost, r.config.Route.TTL)
		log.Ctx(ctx).Debug().Str("key", cacheKey).Any("route", route).Msg("[route] ristrettoRepository.Get get route from Redis successfully")
		routes[key] = route
	}

	return routes, nil
}

func (r *ristrettoRepository) Set(ctx context.Context, keys []valueobject.RouteCacheKeyTTL, routes []*valueobject.SimpleRouteWithExtraData) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "[route] redisCacheRepository.Set")
	defer span.End()

	cachedRoutes, err := r.fallbackRepository.Set(ctx, keys, routes)

	for i, route := range cachedRoutes {
		r.cache.SetWithTTL(genKey(keys[i], r.config.Prefix), route, r.config.Route.Cost, keys[i].TTL)
	}

	return err
}

func (r *ristrettoRepository) Del(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "[route] redisCacheRepository.Del")
	defer span.End()

	err := r.fallbackRepository.Del(ctx, keys)
	if err == nil {
		for _, key := range keys {
			r.cache.Del(genKey(key, r.config.Prefix))
		}
	}

	return err
}
