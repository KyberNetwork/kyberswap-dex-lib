package pool

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"

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

func (r *redisRepository) CheckPoolsInBlacklist(ctx context.Context, addresses []string) ([]bool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.CheckPoolsInBlacklist")
	defer span.End()

	if len(addresses) == 0 {
		return nil, nil
	}

	addresses_interface := lo.Map(addresses, func(address string, _ int) interface{} { return address })
	return r.redisClient.SMIsMember(ctx, r.keyBlacklistedPools, addresses_interface...).Result()
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
				WithFields(logger.Fields{"key": addresses[i]}).
				Warn("invalid pool data")
			continue
		}

		pool, err := decodePool(addresses[i], poolDataStr)
		if err != nil {
			logger.
				WithFields(logger.Fields{"error": err, "key": addresses[i]}).
				Warn("decode pool data failed")
			continue
		}

		pools = append(pools, pool)
	}

	return pools, nil
}
