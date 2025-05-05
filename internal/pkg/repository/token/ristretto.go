package token

import (
	"context"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/dgraph-io/ristretto"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type goCacheRepository struct {
	cache              *ristretto.Cache
	fallbackRepository IFallbackRepository
	config             RistrettoConfig

	// decimal cache which is not need to be expired
	decimalCache *ristretto.Cache
}

func NewGoCacheRepository(
	fallbackRepository IFallbackRepository,
	config RistrettoConfig,
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

	decimalCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.Decimal.NumCounters,
		MaxCost:     config.Decimal.MaxCost,
		BufferItems: config.Decimal.BufferItems,
		Metrics:     true,
	})

	if err != nil {
		return nil, err
	}

	return &goCacheRepository{
		cache:              cache,
		fallbackRepository: fallbackRepository,
		config:             config,
		decimalCache:       decimalCache,
	}, nil
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
		r.cache.SetWithTTL(token.Address, token, r.config.Token.Cost, r.config.Token.TTL)
	}

	return tokens, nil
}

func (r *goCacheRepository) tokenInfoKey(address string) string {
	return utils.Join(KeyTokenInfo, address)
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
	if len(result) != 0 {
		metrics.CountTokenHitLocalCache(ctx, int64(len(result)), true)
	}

	if len(uncachedAddresses) == 0 {
		return result, nil
	}
	metrics.CountTokenHitLocalCache(ctx, int64(len(uncachedAddresses)), false)

	uncachedInfos, err := r.fallbackRepository.FindTokenInfoByAddress(ctx, r.config.ChainID, uncachedAddresses)
	if err != nil {
		return nil, err
	}

	result = append(result, uncachedInfos...)

	for _, info := range result {
		r.cache.Set(r.tokenInfoKey(info.Address), info, r.config.Token.Cost)
	}

	return result, nil
}

func (r *goCacheRepository) FindDecimalByAddresses(ctx context.Context, addresses []string) (map[string]uint8, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[token] goCacheRepository.FindDecimalByAddresses")
	defer span.End()

	decimals := make(map[string]uint8)
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		cachedDecimal, found := r.decimalCache.Get(address)
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		decimal, ok := cachedDecimal.(uint8)
		if !ok {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		decimals[address] = decimal
	}
	if len(decimals) != 0 {
		metrics.CountTokenDecimalHitLocalCache(ctx, int64(len(decimals)), true)
	}

	if len(uncachedAddresses) == 0 {
		return decimals, nil
	}
	metrics.CountTokenDecimalHitLocalCache(ctx, int64(len(uncachedAddresses)), false)

	uncachedTokens, err := r.fallbackRepository.FindByAddresses(ctx, uncachedAddresses)
	if err != nil {
		return nil, err
	}

	for _, token := range uncachedTokens {
		if token == nil {
			continue
		}
		decimals[token.Address] = token.Decimals
	}

	for _, token := range uncachedTokens {
		r.cache.Set(token.Address, token.Decimals, r.config.Decimal.Cost)
	}

	return decimals, nil
}
