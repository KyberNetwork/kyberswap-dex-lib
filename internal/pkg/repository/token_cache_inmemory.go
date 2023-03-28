package repository

import (
	"context"

	"github.com/patrickmn/go-cache"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type TokenCacheRepository struct {
	datastore ITokenDatastoreRepository
	cache     *cache.Cache
}

func NewTokenCacheRepository(
	datastoreRepo ITokenDatastoreRepository,
	cache *cache.Cache,
) *TokenCacheRepository {
	return &TokenCacheRepository{
		datastore: datastoreRepo,
		cache:     cache,
	}
}

func (r *TokenCacheRepository) FindByAddresses(
	ctx context.Context,
	addresses []string,
) ([]entity.Token, error) {
	tokens := make([]entity.Token, 0, len(addresses))
	uncachedAddresses := make([]string, 0, len(addresses))

	for _, address := range addresses {
		token, found := r.cache.Get(address)
		if !found {
			uncachedAddresses = append(uncachedAddresses, address)
			continue
		}

		tokens = append(tokens, token.(entity.Token))
	}

	if len(uncachedAddresses) == 0 {
		return tokens, nil
	}

	uncachedTokens, err := r.datastore.FindByAddresses(ctx, uncachedAddresses)
	if err != nil {
		return nil, err
	}

	for _, token := range uncachedTokens {
		r.cache.Set(token.Address, token, cache.NoExpiration)

		tokens = append(tokens, token)
	}

	return tokens, nil
}
