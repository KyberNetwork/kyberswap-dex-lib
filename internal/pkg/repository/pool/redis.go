package pool

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      RedisRepositoryConfig
	keyPools    string

	keyBlacklistedPools string
	keyFaultyPools      string
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,

		config: config,

		keyPools: utils.Join(config.Prefix, KeyPools),

		keyBlacklistedPools: utils.Join(config.Prefix, KeyBlacklistedPools),

		keyFaultyPools: util.Join(config.Prefix, KeyPools, KeyFaulty),
	}
}

// FindAllAddresses returns all pool addresses
func (r *redisRepository) FindAllAddresses(ctx context.Context) ([]string, error) {
	return r.redisClient.HKeys(ctx, r.keyPools).Result()
}

func (r *redisRepository) GetPoolsInBlacklist(ctx context.Context) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.GetPoolsInBlacklist")
	defer span.End()

	return r.redisClient.SMembers(ctx, r.keyBlacklistedPools).Result()
}

// FindByAddresses returns pools by addresses
func (r *redisRepository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.FindByAddresses")
	defer span.End()

	if len(addresses) == 0 {
		return nil, nil
	}

	poolDataList, err := r.redisClient.HMGet(ctx, r.keyPools, addresses...).Result()
	if err != nil {
		return nil, err
	}

	pools := make([]*entity.Pool, 0, len(poolDataList))
	for i, poolData := range poolDataList {
		if poolData == nil {
			continue
		}

		poolDataStr, ok := poolData.(string)
		if !ok {
			logger.
				WithFields(ctx, logger.Fields{"key": addresses[i]}).
				Warn("invalid pool data")
			continue
		}

		pool, err := decodePool(addresses[i], poolDataStr)
		if err != nil {
			logger.
				WithFields(ctx, logger.Fields{"error": err, "key": addresses[i]}).
				Warn("decode pool data failed")
			continue
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

/*
 * Faulty pools are stored in Redis as a sorted Set.
 * In the sorted set, we set the score as the Unix time at which a pool should expire.
 * To retrieve a list of unexpired faulty pools, we use this command
 * ZRANGE <chainID>:faultyPools (current_unix_timestamp +inf BYSCORE
 **/
func (r *redisRepository) GetFaultyPools(ctx context.Context, startTime, offset, count int64) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.GetFaultyPools")
	defer span.End()

	arg := redis.ZRangeArgs{
		Key:     r.keyFaultyPools,
		Start:   fmt.Sprintf("(%d", startTime),
		Stop:    "+inf",
		ByScore: true,
		Offset:  offset,
		Count:   count,
	}
	return r.redisClient.ZRangeArgs(ctx, arg).Result()
}

func (r *redisRepository) IncreasePoolsTotalCount(ctx context.Context, counter map[string]int64, expiration time.Duration) (map[string]int64, []error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.IncreasePoolsTotalCount")
	defer span.End()

	if len(counter) == 0 {
		return nil, nil
	}
	cmds, err := r.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for key, count := range counter {
			/**
			 * Redis hash key will be chainId:poolAddress:DD:HH:MM
			 * A hash with key format chainId:poolAddress:DD:HH:MM will include 2 fields: KeyTotalCount and failedCount
			 * We will calculate fault density = failedCount/KeyTotalCount
			 */
			hashKey := util.FormatKey(Separator, r.config.Prefix, key)
			/*
			 * HIncreaseBy is an atomic operator which is possible to handle values that may get bigger or smaller depending on the operations performed by the user
			 * In this case, we may get bigger value depends on concurrent requests received by Redis
			 * https://redis.io/commands/hincrby/
			 *
			 * Adding chainId to the hash key, key format: chainId:poolAddress:DD:HH:MM, hash field format: KeyTotalCount
			 **/
			pipe.HIncrBy(ctx, hashKey, KeyTotalCount, count)
			/*
			 * Currently, we always expire keys although the keys has already been set expired time
			 * Because of race conditions, we can't check return value in IncrBy to set expired time
			 * Instead, we can use lua script, refer link: https://redis.io/commands/incr/
			 * We will improve later if setting expire as always harms Redis performance
			 * NX -- Set expiry only when the key has no expiry
			 **/
			pipe.ExpireNX(ctx, hashKey, expiration)
		}
		return nil
	})
	if err != nil {
		return nil, []error{err}
	}

	results := make(map[string]int64)
	errors := make([]error, 0, len(counter))
	for _, cmd := range cmds {
		if cmd, ok := cmd.(*redis.IntCmd); ok {
			value, err := cmd.Result()
			// remove prefix chainId
			key := cmd.Args()[1].(string)[(len(r.config.Prefix) + 1):]
			if err != nil {
				errors = append(errors, fmt.Errorf("[CountTotalPools] failed hash key: %s, error: %v", key, err))
			} else {
				results[key] = value
			}
		}
	}

	return results, errors
}
