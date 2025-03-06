package route

import (
	"context"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/redis/go-redis/v9"
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

func (r *redisRepository) Save(ctx context.Context, routeId string, alphaFee *entity.AlphaFee) error {
	data, err := encodeAlphaFee(alphaFee)
	if err != nil {
		logger.WithFields(ctx, logger.Fields{"error": err}).Errorf("Encode alphaFee error")
		return err
	}

	return r.redisClient.Set(ctx, formatKey(r.config.Redis.Separator, r.config.Redis.Prefix, KeyAlphaFee, routeId), data, r.config.Redis.TTL).Err()
}

func (r *redisRepository) GetByRouteId(ctx context.Context, routeId string) (*entity.AlphaFee, error) {
	data, err := r.redisClient.Get(ctx, formatKey(r.config.Redis.Separator, r.config.Redis.Prefix, KeyAlphaFee, routeId)).Result()
	if err != nil {
		return nil, err
	}

	return decodeAlphaFee(data)
}

func encodeAlphaFee(alphaFee *entity.AlphaFee) (string, error) {
	bytes, err := json.Marshal(alphaFee)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodeAlphaFee(data string) (*entity.AlphaFee, error) {
	var alphaFee entity.AlphaFee
	if err := json.Unmarshal([]byte(data), &alphaFee); err != nil {
		return nil, err
	}

	return &alphaFee, nil
}
