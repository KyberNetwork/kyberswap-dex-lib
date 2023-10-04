package pool

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      RedisRepositoryConfig
	keyPools    string

	keyBlacklistedPools string
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,

		config: config,

		keyPools: utils.Join(config.Prefix, KeyPools),

		keyBlacklistedPools: utils.Join(config.Prefix, KeyBlacklistedPools),
	}
}

// FindAllAddresses returns all pool addresses
func (r *redisRepository) FindAllAddresses(ctx context.Context) ([]string, error) {
	return r.redisClient.HKeys(ctx, r.keyPools).Result()
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
