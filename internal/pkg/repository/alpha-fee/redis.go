package route

import (
	"context"
	"strings"

	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      Config
}

func NewRedisRepository(redisClient redis.UniversalClient, config Config) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,
		config:      config,
	}
}

func formatKey(seperator string, args ...string) string {
	return strings.Join(args, seperator)
}

func (r *redisRepository) Save(ctx context.Context, routeId string, alphaFee *entity.AlphaFeeV2) error {
	data, err := encodeAlphaFee(alphaFee)
	if err != nil {
		logger.WithFields(ctx, logger.Fields{"error": err}).Errorf("Encode alphaFee error")
		return err
	}

	return r.redisClient.Set(ctx, formatKey(r.config.Redis.Separator, r.config.Redis.Prefix, KeyAlphaFeeV2, routeId),
		data, r.config.Redis.TTL).Err()
}

func (r *redisRepository) GetByRouteId(ctx context.Context, routeId string) (*entity.AlphaFeeV2, error) {
	data, err := r.redisClient.Get(ctx,
		formatKey(r.config.Redis.Separator, r.config.Redis.Prefix, KeyAlphaFeeV2, routeId)).Result()
	if err != nil {
		return nil, err
	}

	return decodeAlphaFee(data)
}

func encodeAlphaFee(alphaFee *entity.AlphaFeeV2) (string, error) {
	bytes, err := json.Marshal(alphaFee)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodeAlphaFee(data string) (*entity.AlphaFeeV2, error) {
	var alphaFee entity.AlphaFeeV2
	if err := json.Unmarshal([]byte(data), &alphaFee); err != nil {
		return nil, err
	}

	return &alphaFee, nil
}
