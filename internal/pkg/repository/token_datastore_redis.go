package repository

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

type TokenDatastoreRedisRepository struct {
	db *redis.Redis
}

func NewTokenDataStoreRedisRepository(
	db *redis.Redis,
) *TokenDatastoreRedisRepository {
	return &TokenDatastoreRedisRepository{
		db: db,
	}
}

func (r *TokenDatastoreRedisRepository) FindAll(
	ctx context.Context,
) ([]entity.Token, error) {
	tokenMap, err := r.db.Client.HGetAll(
		ctx,
		r.db.FormatKey(entity.TokenKey),
	).Result()

	if err != nil {
		return nil, err
	}

	tokens := make([]entity.Token, 0, len(tokenMap))

	for key, tokenString := range tokenMap {
		tokens = append(tokens, entity.DecodeToken(key, tokenString))
	}

	return tokens, nil
}

func (r *TokenDatastoreRedisRepository) FindByAddresses(
	ctx context.Context,
	addresses []string,
) ([]entity.Token, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	tokenStrings, err := r.db.Client.HMGet(
		ctx,
		r.db.FormatKey(entity.TokenKey),
		addresses...,
	).Result()

	if err != nil {
		return nil, err
	}

	tokens := make([]entity.Token, 0, len(tokenStrings))

	for i, poolString := range tokenStrings {
		if poolString != nil {
			tokens = append(
				tokens,
				entity.DecodeToken(addresses[i], poolString.(string)),
			)
		}
	}

	return tokens, nil
}

func (r *TokenDatastoreRedisRepository) Persist(
	ctx context.Context,
	token entity.Token,
) error {
	_, err := r.db.Client.HSet(
		ctx,
		r.db.FormatKey(entity.TokenKey),
		token.Address,
		token.Encode(),
	).Result()

	return err
}

func (r *TokenDatastoreRedisRepository) Delete(
	ctx context.Context,
	token entity.Token,
) error {
	_, err := r.db.Client.HDel(
		ctx,
		r.db.FormatKey(entity.TokenKey),
		token.Address,
	).Result()

	return err
}
