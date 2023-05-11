package token

import (
	"context"

	"github.com/patrickmn/go-cache"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
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
	defer span.Finish()

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
