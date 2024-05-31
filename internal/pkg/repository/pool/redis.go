package pool

import (
	"context"
	"errors"

	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"

	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var faultyPoolKey = "faultyPools"

type redisRepository struct {
	redisClient redis.UniversalClient
	config      Config
	keyPools    string

	keyBlacklistedPools string
	keyFaultyPools      string

	poolClient IPoolClient

	// local cache to cache only faulty pools
	cache *ristretto.Cache
}

func NewRedisRepository(redisClient redis.UniversalClient, poolClient IPoolClient, config Config) (*redisRepository, error) {

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.Ristretto.NumCounters,
		MaxCost:     config.Ristretto.MaxCost,
		BufferItems: config.Ristretto.BufferItems,
	})
	if err != nil {
		return nil, err
	}

	return &redisRepository{
		redisClient: redisClient,
		poolClient:  poolClient,

		config: config,

		keyPools: utils.Join(config.Redis.Prefix, KeyPools),

		keyBlacklistedPools: utils.Join(config.Redis.Prefix, KeyBlacklistedPools),

		keyFaultyPools: util.Join(config.Redis.Prefix, KeyPools, KeyFaulty),

		cache: cache,
	}, nil
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
func (r *redisRepository) GetFaultyPools(ctx context.Context) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.GetFaultyPools")
	defer span.End()
	if r.config.Redis.MaxFaultyPoolSize <= 0 {
		return []string{}, errors.New("config MaxFaultyPoolSize must be postitive")
	}
	// we accept cached data can be temporarily wrong during TTL time (30s) to reduce number of request in pool-service
	// faulty list will be reset after 30s

	cachedKey := utils.Join(r.config.Redis.Prefix, faultyPoolKey)
	if cachedData, ok := r.cache.Get(cachedKey); ok {
		if addresses, ok := cachedData.([]string); ok {
			return addresses, nil
		}
	}

	offset := int64(0)
	result := []string{}
	for {
		faultyPools, err := r.poolClient.GetFaultyPools(ctx, offset, r.config.Redis.MaxFaultyPoolSize)
		if err != nil {
			return []string{}, nil
		}

		result = append(result, faultyPools...)

		// if faulty pool size is smaller than max config, then we already got the whole list
		// adding len(faultyPools) == 0 to easy unit test (because MaxFaultyPoolSize is usually configured to 0 when unit test)
		if len(faultyPools) == 0 || int64(len(faultyPools)) < r.config.Redis.MaxFaultyPoolSize {
			break
		}
		offset += r.config.Redis.MaxFaultyPoolSize
	}

	r.cache.SetWithTTL(cachedKey, result, 1, r.config.Ristretto.FaultyPools.TTL)

	return result, nil

}

func (r *redisRepository) TrackFaultyPools(ctx context.Context, trackers []routerEntities.FaultyPoolTracker) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.TrackFaultyPools")
	defer span.End()

	return r.poolClient.TrackFaultyPools(ctx, trackers)
}
