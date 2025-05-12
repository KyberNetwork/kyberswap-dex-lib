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

type IToken interface {
	GetAddress() string
}

type repository[T IToken] struct {
	redisClient redis.UniversalClient
	httpClient  ITokenAPI
	config      RedisRepositoryConfig
	keyTokens   string
}

func NewSimplifiedTokenRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig, tokenAPI ITokenAPI) *repository[entity.SimplifiedToken] {
	return &repository[entity.SimplifiedToken]{
		redisClient: redisClient,
		config:      config,
		keyTokens:   utils.Join(config.Prefix, KeyTokens),
		httpClient:  tokenAPI,
	}
}

func NewFullTokenRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig, tokenAPI ITokenAPI) *repository[entity.Token] {
	return &repository[entity.Token]{
		redisClient: redisClient,
		config:      config,
		keyTokens:   utils.Join(config.Prefix, KeyTokens),
		httpClient:  tokenAPI,
	}
}

// FindByAddresses returns tokens by their addresses
func (r *repository[T]) FindByAddresses(ctx context.Context, addresses []string) ([]*T, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "token.repository.FindByAddresses")
	defer span.End()

	if len(addresses) == 0 {
		return nil, nil
	}

	tokenDataList, err := r.redisClient.HMGet(ctx, r.keyTokens, addresses...).Result()
	if err != nil {
		return nil, err
	}

	tokens := make([]*T, 0, len(tokenDataList))
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

		token, err := decodeToken[T](ctx, tokenDataStr, addresses[i])
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

func (r *repository[T]) FindTokenInfoByAddress(ctx context.Context, chainID valueobject.ChainID, addresses []string) ([]*routerEntity.TokenInfo, error) {
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
