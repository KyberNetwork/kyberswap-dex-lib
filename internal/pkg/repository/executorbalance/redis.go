package executorbalance

import (
	"context"
	"strconv"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

type RedisRepository struct {
	redisClient        redis.UniversalClient
	tokenBalancePrefix string
	poolApprovalPrefix string
	blockNumberPrefix  string
}

const (
	KeyToken             = "executor-token-balance"
	KeyPoolApproval      = "executor-pool-approval"
	KeyLatestBlockNumber = "executor-block-number"
)

func NewRedisRepository(
	redisClient redis.UniversalClient,
	config Config,
) *RedisRepository {
	return &RedisRepository{
		redisClient: redisClient,

		tokenBalancePrefix: utils.Join(config.Prefix, KeyToken),
		poolApprovalPrefix: utils.Join(config.Prefix, KeyPoolApproval),
		blockNumberPrefix:  utils.Join(config.Prefix, KeyLatestBlockNumber),
	}
}

func (r *RedisRepository) HasToken(ctx context.Context, executorAddress string, queries []string) ([]bool, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	key := utils.Join(r.tokenBalancePrefix, executorAddress)
	members := lo.Map(queries, func(query string, _ int) interface{} { return query })

	result, err := r.redisClient.SMIsMember(ctx, key, members...).Result()

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *RedisRepository) HasPoolApproval(ctx context.Context, executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	cmds, err := r.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, query := range queries {
			key := utils.Join(r.poolApprovalPrefix, executorAddress, query.TokenIn)
			pipe.SIsMember(ctx, key, query.PoolAddress)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	result := make([]bool, 0, len(queries))
	for _, cmd := range cmds {
		result = append(result, cmd.(*redis.BoolCmd).Val())
	}

	return result, nil
}

func (r *RedisRepository) AddToken(ctx context.Context, executorAddress string, data []string) error {
	if len(data) == 0 {
		return nil
	}

	key := utils.Join(r.tokenBalancePrefix, executorAddress)
	members := lo.Map(data, func(query string, _ int) interface{} { return query })

	_, err := r.redisClient.SAdd(ctx, key, members...).Result()
	return err
}

func (r *RedisRepository) RemoveToken(ctx context.Context, executorAddress string, data []string) error {
	if len(data) == 0 {
		return nil
	}

	key := utils.Join(r.tokenBalancePrefix, executorAddress)
	members := lo.Map(data, func(query string, _ int) interface{} { return query })

	_, err := r.redisClient.SRem(ctx, key, members...).Result()
	return err
}

func (r *RedisRepository) ApprovePool(ctx context.Context, executorAddress string, data []dto.PoolApprovalQuery) error {
	if len(data) == 0 {
		return nil
	}

	_, err := r.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, query := range data {
			key := utils.Join(r.poolApprovalPrefix, executorAddress, query.TokenIn)
			pipe.SAdd(ctx, key, query.PoolAddress)
		}
		return nil
	})
	return err
}

func (r *RedisRepository) CleanUp(ctx context.Context, executorAddress string) error {
	// TODO: Low priority feature, will implement later.
	return nil
}

func (r *RedisRepository) GetLatestProcessedBlockNumber(ctx context.Context, executorAddress string) (uint64, error) {
	key := utils.Join(r.blockNumberPrefix, executorAddress)
	blockNumberString, err := r.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	return strconv.ParseUint(blockNumberString, 10, 64)
}

func (r *RedisRepository) UpdateLatestProcessedBlockNumber(ctx context.Context, executorAddress string, blockNumber uint64) error {
	key := utils.Join(r.blockNumberPrefix, executorAddress)
	_, err := r.redisClient.Set(ctx, key, blockNumber, 0).Result()
	return err
}
