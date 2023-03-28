package repository

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type TokenRepository struct {
	ITokenDatastoreRepository
	ITokenCacheRepository
}

func NewTokenRepository(
	datastoreRepo ITokenDatastoreRepository,
	cacheRepo ITokenCacheRepository,
) *TokenRepository {
	return &TokenRepository{
		ITokenDatastoreRepository: datastoreRepo,
		ITokenCacheRepository:     cacheRepo,
	}
}

func (r *TokenRepository) Save(ctx context.Context, token entity.Token) error {
	if err := r.Persist(ctx, token); err != nil {
		return err
	}

	if err := r.Set(ctx, token.Address, token); err != nil {
		return err
	}

	return nil
}
