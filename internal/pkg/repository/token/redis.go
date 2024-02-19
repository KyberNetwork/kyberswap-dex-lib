package token

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
	keyTokens   string
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,
		config:      config,
		keyTokens:   utils.Join(config.Prefix, KeyTokens),
	}
}

// FindByAddresses returns tokens by their addresses
func (r *redisRepository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[token] redisRepository.FindByAddresses")
	defer span.End()

	if len(addresses) == 0 {
		return nil, nil
	}

	tokenDataList, err := r.redisClient.HMGet(ctx, r.keyTokens, addresses...).Result()
	if err != nil {
		return nil, err
	}

	tokens := make([]*entity.Token, 0, len(tokenDataList))
	for i, tokenData := range tokenDataList {
		if tokenData == nil {
			continue
		}

		tokenDataStr, ok := tokenData.(string)
		if !ok {
			logger.
				WithFields(ctx, logger.Fields{"key": addresses[i]}).
				Warn("invalid token data")
			continue
		}

		token, err := decodeToken(addresses[i], tokenDataStr)
		if err != nil {
			logger.
				WithFields(ctx, logger.Fields{"error": err, "key": addresses[i]}).
				Warn("decode token data failed")
			continue
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}
