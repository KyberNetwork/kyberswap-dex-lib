package l2fee

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/redis/go-redis/v9"
)

const (
	KeyMetadata      = "metadata"
	FieldL1FeeParams = "l1_fee_params"
)

type (
	RedisL1FeeRepositoryConfig struct {
		Prefix string `mapstructure:"prefix"`
	}

	RedisL1FeeRepository struct {
		redisClient redis.UniversalClient
		config      RedisL1FeeRepositoryConfig
		keyMetadata string
	}
)

func NewRedisRepository(
	redisClient redis.UniversalClient,
	config RedisL1FeeRepositoryConfig,
) *RedisL1FeeRepository {
	return &RedisL1FeeRepository{
		redisClient: redisClient,
		config:      config,
		keyMetadata: utils.Join(config.Prefix, KeyMetadata),
	}
}

func (r *RedisL1FeeRepository) UpdateL1FeeParams(ctx context.Context, params any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	err = r.redisClient.HSet(ctx, r.keyMetadata, FieldL1FeeParams, string(data)).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisL1FeeRepository) GetL1FeeParams(ctx context.Context, output any) error {
	paramsStr, err := r.redisClient.HGet(ctx, r.keyMetadata, FieldL1FeeParams).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(paramsStr), output)
}
