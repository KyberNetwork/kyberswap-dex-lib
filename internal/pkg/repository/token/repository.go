package token

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type repository struct {
	redisClient redis.UniversalClient
	httpClient  ITokenAPI
	config      RedisRepositoryConfig
	keyTokens   string
}

func NewRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig, tokenAPI ITokenAPI) *repository {
	return &repository{
		redisClient: redisClient,
		config:      config,
		keyTokens:   utils.Join(config.Prefix, KeyTokens),
		httpClient:  tokenAPI,
	}
}

// FindByAddresses returns tokens by their addresses
func (r *repository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "token.repository.FindByAddresses")
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

func (r *repository) FindTokenInfoByAddress(ctx context.Context, chainID valueobject.ChainID, addresses []string) ([]*routerEntity.TokenInfo, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "token.repository.FindTokenInfoByAddress")
	defer span.End()

	if len(addresses) == 0 {
		return nil, nil
	}

	tokenInfoList, err := r.httpClient.FindTokenInfos(ctx, chainID, addresses)
	if err != nil {
		return nil, err
	}

	return tokenInfoList, nil
}
