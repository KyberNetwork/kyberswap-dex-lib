package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"

	cmap "github.com/orcaman/concurrent-map"
)

var ErrTokenNotFoundInCache = errors.New("can not get token in cache")

type TokenCacheCMapRepository struct {
	tokenMap cmap.ConcurrentMap
}

func NewTokenCacheCMapRepository(cache cmap.ConcurrentMap) *TokenCacheCMapRepository {
	return &TokenCacheCMapRepository{
		tokenMap: cache,
	}
}

func (r *TokenCacheCMapRepository) Keys(_ context.Context) []string {
	return r.tokenMap.Keys()
}

func (r *TokenCacheCMapRepository) Get(_ context.Context, address string) (entity.Token, error) {
	token, ok := r.tokenMap.Get(address)
	if !ok {
		return entity.Token{
			Address: address,
		}, ErrTokenNotFoundInCache
	}

	return token.(entity.Token), nil
}

func (r *TokenCacheCMapRepository) Set(_ context.Context, address string, token entity.Token) error {
	r.tokenMap.Set(address, token)
	return nil
}

func (r *TokenCacheCMapRepository) Remove(_ context.Context, address string) error {
	r.tokenMap.Remove(address)
	return nil
}

func (r *TokenCacheCMapRepository) Count(_ context.Context) int {
	return r.tokenMap.Count()
}

func (r *TokenCacheCMapRepository) GetByAddresses(_ context.Context, ids []string) ([]entity.Token, error) {
	var tokens []entity.Token

	for _, id := range ids {
		pool, ok := r.tokenMap.Get(id)
		if !ok {
			return []entity.Token{}, fmt.Errorf("%w: %s", ErrTokenNotFoundInCache, id)
		}

		tokens = append(tokens, pool.(entity.Token))
	}

	return tokens, nil
}
