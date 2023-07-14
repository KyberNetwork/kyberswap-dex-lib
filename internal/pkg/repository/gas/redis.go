package gas

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	gasPricer   ethereum.GasPricer

	config RedisRepositoryConfig

	keyMetadata string
}

func NewRedisRepository(
	redisClient redis.UniversalClient,
	gasPricer ethereum.GasPricer,
	config RedisRepositoryConfig,
) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,
		gasPricer:   gasPricer,
		config:      config,
		keyMetadata: utils.Join(config.Prefix, KeyMetadata),
	}
}

// UpdateSuggestedGasPrice update latest suggested gas price
func (r *redisRepository) UpdateSuggestedGasPrice(ctx context.Context) (*big.Int, error) {
	suggestedGasPrice, err := r.gasPricer.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	err = r.redisClient.HSet(ctx, r.keyMetadata, FieldSuggestedGasPrice, suggestedGasPrice.String()).Err()
	if err != nil {
		return nil, err
	}

	return suggestedGasPrice, nil
}

// GetSuggestedGasPrice returns current suggested gas price
func (r *redisRepository) GetSuggestedGasPrice(ctx context.Context) (*big.Int, error) {
	suggestedGasPriceStr, err := r.redisClient.HGet(ctx, r.keyMetadata, FieldSuggestedGasPrice).Result()
	if err != nil {
		return nil, err
	}

	suggestedGasPrice, isValid := new(big.Int).SetString(suggestedGasPriceStr, 10)
	if !isValid {
		return nil, ErrInvalidGasPrice
	}
	return suggestedGasPrice, nil
}
