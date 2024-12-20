package token

import (
	"context"
	"strings"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/patrickmn/go-cache"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type goCacheRepository struct {
	cache              *cache.Cache
	fallbackRepository IFallbackRepository
	config             GoCacheRepositoryConfig
}

func NewGoCacheRepository(
	fallbackRepository IFallbackRepository,
	config GoCacheRepositoryConfig,
) *goCacheRepository {
	return &goCacheRepository{
		cache:              cache.New(config.Expiration, config.CleanupInterval),
		fallbackRepository: fallbackRepository,
		config:             config,
	}
}

// FindByAddresses looks for token in cache, if the token is not cached, find it from fallbackRepository and cache them
func (r *goCacheRepository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[token] goCacheRepository.FindByAddresses")
	defer span.End()

	tokens := make([]*entity.Token, 0, len(addresses))
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		cachedToken, found := r.cache.Get(address)
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		token, ok := cachedToken.(*entity.Token)
		if !ok {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		tokens = append(tokens, token)
	}

	if len(uncachedAddresses) == 0 {
		return tokens, nil
	}

	uncachedTokens, err := r.fallbackRepository.FindByAddresses(ctx, uncachedAddresses)
	if err != nil {
		return nil, err
	}

	tokens = append(tokens, uncachedTokens...)

	for _, token := range uncachedTokens {
		r.cache.Set(token.Address, token, r.config.Expiration)
	}

	return tokens, nil
}

func (r *goCacheRepository) tokenInfoKey(address string) string {
	return strings.Join([]string{KeyTokenInfo, address}, utils.SliceParamsItemSeparator)
}

func (r *goCacheRepository) FindTokenInfoByAddress(ctx context.Context, addresses []string) ([]*routerEntity.TokenInfo, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[token] redisRepository.FindTokenInfoByAddress")
	defer span.End()

	result := make([]*routerEntity.TokenInfo, 0, len(addresses))
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		cachedToken, found := r.cache.Get(r.tokenInfoKey(address))
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		token, ok := cachedToken.(*routerEntity.TokenInfo)
		if !ok {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		result = append(result, token)
	}

	if len(uncachedAddresses) == 0 {
		return result, nil
	}

	uncachedInfos, err := r.fallbackRepository.FindTokenInfoByAddress(ctx, r.config.ChainID, uncachedAddresses)
	if err != nil {
		return nil, err
	}

	result = append(result, uncachedInfos...)

	for _, info := range result {
		r.cache.Set(r.tokenInfoKey(info.Address), info, r.config.Expiration)
	}

	return result, nil
}
