package route

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      RedisRepositoryConfig
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,
		config:      config,
	}
}

func (r *redisRepository) Get(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) (map[valueobject.RouteCacheKeyTTL]*valueobject.SimpleRoute, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[route] redisRepository.Get")
	defer span.End()

	if len(keys) == 0 {
		return nil, nil
	}

	redisKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		redisKeys = append(redisKeys, genKey(k, r.config.Prefix))
	}

	routeData, err := r.redisClient.MGet(ctx, redisKeys...).Result()

	if err != nil {
		return nil, err
	}

	results := map[valueobject.RouteCacheKeyTTL]*valueobject.SimpleRoute{}
	for i, data := range routeData {
		if data == nil {
			continue
		}

		routeDataStr, ok := data.(string)
		if !ok {
			logger.WithFields(ctx, logger.Fields{"data": routeDataStr, "key": redisKeys[i]}).Errorf("data is not a string")
			continue
		}

		route, err := decodeRoute(routeDataStr)

		if err != nil {
			logger.WithFields(ctx, logger.Fields{"data": routeDataStr, "key": redisKeys[i]}).Errorf("invalid route data in Redis")
			continue
		}

		results[keys[i]] = route
	}

	return results, nil
}

func (r *redisRepository) Set(ctx context.Context, keys []valueobject.RouteCacheKeyTTL, routes []*valueobject.SimpleRoute) ([]*valueobject.SimpleRoute, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[route] redisRepository.Set")
	defer span.End()

	if len(keys) == 0 {
		return nil, nil
	}

	cmds, e := r.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for i, key := range keys {
			encoded, err := encodeRoute(*routes[i])
			if err != nil {
				logger.WithFields(ctx, logger.Fields{"error": err}).Errorf("Encode route error")
				continue
			}
			pipe.Set(ctx, genKey(key, r.config.Prefix), encoded, key.TTL)
		}
		return nil
	})

	if e != nil {
		return nil, e
	}

	results := make([]*valueobject.SimpleRoute, 0, len(cmds))
	var err error
	for i, cmd := range cmds {
		if cmd, ok := cmd.(*redis.StatusCmd); ok {
			if _, e := cmd.Result(); e != nil {
				err = fmt.Errorf("[route] redisRepository.Set failed key: %s, error: %v", cmd.Args()[1].(string), e)
			} else {
				results = append(results, routes[i])
			}
		}
	}

	return results, err
}

func (r *redisRepository) Del(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "[route] redisRepository.Del")
	defer span.End()

	if len(keys) == 0 {
		return nil
	}

	redisKeys := []string{}
	for _, k := range keys {
		redisKeys = append(redisKeys, genKey(k, r.config.Prefix))
	}

	return r.redisClient.Del(ctx, redisKeys...).Err()

}
