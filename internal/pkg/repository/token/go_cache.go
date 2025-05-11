package token

import (
	"context"
	"time"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/dgraph-io/ristretto"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type goCacheRepository struct {
	cache              *ristretto.Cache
	fallbackRepository IFallbackRepository[entity.SimplifiedToken]
	config             *RistrettoConfig

	tokenInfoCache *ristretto.Cache
}

func NewGoCacheRepository(
	fallbackRepository IFallbackRepository[entity.SimplifiedToken],
	config *RistrettoConfig,
) (*goCacheRepository, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.Token.NumCounters,
		MaxCost:     config.Token.MaxCost,
		BufferItems: config.Token.BufferItems,
		Metrics:     true,
	})

	if err != nil {
		return nil, err
	}

	tokenInfoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.TokenInfo.NumCounters,
		MaxCost:     config.TokenInfo.MaxCost,
		BufferItems: config.TokenInfo.BufferItems,
		Metrics:     true,
	})

	if err != nil {
		return nil, err
	}

	return &goCacheRepository{
		cache:              cache,
		fallbackRepository: fallbackRepository,
		config:             config,
		tokenInfoCache:     tokenInfoCache,
	}, nil
}

// FindByAddresses looks for token in cache, if the token is not cached, find it from fallbackRepository and cache them
func (r *goCacheRepository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.SimplifiedToken, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[token] goCacheRepository.FindByAddresses")
	defer span.End()

	tokens := make([]*entity.SimplifiedToken, 0, len(addresses))
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		cachedToken, found := r.cache.Get(address)
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		token, ok := cachedToken.(*entity.SimplifiedToken)
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
		r.cache.SetWithTTL(token.Address, token, r.config.Token.Cost, r.getTokenTTL(token.Address))
	}

	return tokens, nil
}

func (r *goCacheRepository) getTokenTTL(address string) time.Duration {
	if r.config.WhitelistedTokenSet[address] {
		return time.Duration(0)
	}

	return r.config.Token.TTL
}

func (r *goCacheRepository) FindTokenInfoByAddress(ctx context.Context, addresses []string) ([]*routerEntity.TokenInfo, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[token] redisRepository.FindTokenInfoByAddress")
	defer span.End()

	result := make([]*routerEntity.TokenInfo, 0, len(addresses))
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		cachedInfo, found := r.tokenInfoCache.Get(address)
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		token, ok := cachedInfo.(*routerEntity.TokenInfo)
		if !ok {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		result = append(result, token)
	}
	if len(result) != 0 {
		metrics.CountTokenInfoHitLocalCache(ctx, int64(len(result)), true)
	}

	if len(uncachedAddresses) == 0 {
		return result, nil
	}
	metrics.CountTokenInfoHitLocalCache(ctx, int64(len(uncachedAddresses)), false)

	uncachedInfos, err := r.fallbackRepository.FindTokenInfoByAddress(ctx, r.config.ChainID, uncachedAddresses)
	if err != nil {
		return nil, err
	}

	result = append(result, uncachedInfos...)

	for _, info := range result {
		// We do not retrive token info for whitelist token, so do not need to modify TTL with whitelist token set
		r.tokenInfoCache.SetWithTTL(info.Address, info, r.config.Token.Cost, r.config.Token.TTL)
	}

	return result, nil
}
