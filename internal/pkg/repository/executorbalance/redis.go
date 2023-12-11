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

func (r *RedisRepository) HasToken(executorAddress string, queries []string) ([]bool, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	key := utils.Join(r.tokenBalancePrefix, executorAddress)
	members := lo.Map(queries, func(query string, _ int) interface{} { return query })

	result, err := r.redisClient.SMIsMember(context.Background(), key, members...).Result()

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *RedisRepository) HasPoolApproval(executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	cmds, err := r.redisClient.Pipelined(context.Background(), func(pipe redis.Pipeliner) error {
		for _, query := range queries {
			key := utils.Join(r.poolApprovalPrefix, executorAddress, query.TokenIn)
			pipe.SIsMember(context.Background(), key, query.PoolAddress)
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

func (r *RedisRepository) AddToken(executorAddress string, data []string) error {
	if len(data) == 0 {
		return nil
	}

	key := utils.Join(r.tokenBalancePrefix, executorAddress)
	members := lo.Map(data, func(query string, _ int) interface{} { return query })

	_, err := r.redisClient.SAdd(context.Background(), key, members...).Result()
	return err
}

func (r *RedisRepository) ApprovePool(executorAddress string, data []dto.PoolApprovalQuery) error {
	if len(data) == 0 {
		return nil
	}

	_, err := r.redisClient.Pipelined(context.Background(), func(pipe redis.Pipeliner) error {
		for _, query := range data {
			key := utils.Join(r.poolApprovalPrefix, executorAddress, query.TokenIn)
			pipe.SAdd(context.Background(), key, query.PoolAddress)
		}
		return nil
	})
	return err
}

func (r *RedisRepository) CleanUp(executorAddress string) error {
	// TODO: Low priority feature, will implement later.
	return nil
}

func (r *RedisRepository) GetLatestProcessedBlockNumber(executorAddress string) (uint64, error) {
	key := utils.Join(r.blockNumberPrefix, executorAddress)
	blockNumberString, err := r.redisClient.Get(context.Background(), key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	return strconv.ParseUint(blockNumberString, 10, 64)
}

func (r *RedisRepository) UpdateLatestProcessedBlockNumber(executorAddress string, blockNumber uint64) error {
	key := utils.Join(r.blockNumberPrefix, executorAddress)
	_, err := r.redisClient.Set(context.Background(), key, blockNumber, 0).Result()
	return err
}
