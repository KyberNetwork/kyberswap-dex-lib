package price

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      RedisRepositoryConfig
	keyPrices   string
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,
		config:      config,
		keyPrices:   utils.Join(config.Prefix, KeyPrices),
	}
}

// FindByAddresses returns prices from token addresses
func (r *redisRepository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[price] redisRepository.FindByAddresses")
	defer span.End()

	if len(addresses) == 0 {
		return nil, nil
	}

	priceDataList, err := r.redisClient.HMGet(ctx, r.keyPrices, addresses...).Result()
	if err != nil {
		return nil, err
	}

	prices := make([]*entity.Price, 0, len(priceDataList))
	for i, priceData := range priceDataList {
		if priceData == nil {
			continue
		}

		priceDataStr, ok := priceData.(string)
		if !ok {
			logger.
				WithFields(ctx, logger.Fields{"key": addresses[i]}).
				Warn("invalid price data")
			continue
		}

		price, err := decodePrice(addresses[i], priceDataStr)
		if err != nil {
			logger.
				WithFields(ctx, logger.Fields{"error": err, "key": addresses[i]}).
				Warn("decode price data failed")
			continue
		}

		prices = append(prices, price)
	}

	return prices, nil
}
