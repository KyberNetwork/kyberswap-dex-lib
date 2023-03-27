package repository

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RouteCacheRedisRepository struct {
	readClient    *redis.Client
	writeClient   *redis.Client
	clusterClient *redis.ClusterClient
}

func NewRouteCacheRedisRepository(
	readClient *redis.Client,
	writeClient *redis.Client,
	clusterClient *redis.ClusterClient,
) *RouteCacheRedisRepository {
	return &RouteCacheRedisRepository{
		readClient:    readClient,
		writeClient:   writeClient,
		clusterClient: clusterClient,
	}
}

func (r *RouteCacheRedisRepository) Set(ctx context.Context, key string, data string, ttl time.Duration) error {
	if r.clusterClient != nil {
		return r.clusterClient.
			Set(ctx, key, []byte(data), ttl).
			Err()
	}

	return r.writeClient.
		Set(ctx, key, []byte(data), ttl).
		Err()
}

func (r *RouteCacheRedisRepository) Get(ctx context.Context, key string) ([]byte, time.Duration, error) {
	if r.clusterClient != nil {
		return r.getFromClusterRedis(ctx, key)
	}

	return r.getFromMainRedis(ctx, key)
}

func (r *RouteCacheRedisRepository) getFromClusterRedis(ctx context.Context, key string) ([]byte, time.Duration, error) {
	cmds, err := r.clusterClient.Pipelined(
		ctx, func(pipe redis.Pipeliner) error {
			if err := pipe.Get(ctx, key).Err(); err != nil {
				return err
			}

			if err := pipe.TTL(ctx, key).Err(); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, 0, err
	}

	data, err := cmds[0].(*redis.StringCmd).Bytes()
	if err != nil {
		return nil, 0, err
	}

	ttl, err := cmds[1].(*redis.DurationCmd).Result()
	if err != nil {
		return nil, 0, err
	}

	return data, ttl, err
}

func (r *RouteCacheRedisRepository) getFromMainRedis(ctx context.Context, key string) ([]byte, time.Duration, error) {
	cmds, err := r.readClient.Pipelined(
		ctx, func(pipe redis.Pipeliner) error {
			if err := pipe.Get(ctx, key).Err(); err != nil {
				return err
			}

			if err := pipe.TTL(ctx, key).Err(); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, 0, err
	}

	data, err := cmds[0].(*redis.StringCmd).Bytes()
	if err != nil {
		return nil, 0, err
	}

	ttl, err := cmds[1].(*redis.DurationCmd).Result()
	if err != nil {
		return nil, 0, err
	}

	return data, ttl, err
}
