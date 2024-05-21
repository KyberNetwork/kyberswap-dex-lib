package pool

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"

	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      RedisRepositoryConfig
	keyPools    string

	keyBlacklistedPools string
	keyFaultyPools      string

	poolClient IPoolClient
}

func NewRedisRepository(redisClient redis.UniversalClient, poolClient IPoolClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,
		poolClient:  poolClient,

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
func (r *redisRepository) GetFaultyPools(ctx context.Context, offset, count int64) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.GetFaultyPools")
	defer span.End()

	return r.poolClient.GetFaultyPools(ctx, offset, count)
}

func (r *redisRepository) TrackFaultyPools(ctx context.Context, trackers []routerEntities.FaultyPoolTracker) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.TrackFaultyPools")
	defer span.End()

	return r.poolClient.TrackFaultyPools(ctx, trackers)
}
