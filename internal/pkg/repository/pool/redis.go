package pool

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      RedisRepositoryConfig
	keyPools    string
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,

		config: config,

		keyPools: utils.Join(config.Prefix, KeyPools),
	}
}

// FindAllAddresses returns all pool addresses
func (r *redisRepository) FindAllAddresses(ctx context.Context) ([]string, error) {
	return r.redisClient.HKeys(ctx, r.keyPools).Result()
}

// FindByAddresses returns pools by addresses
func (r *redisRepository) FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[pool] redisRepository.FindByAddresses")
	defer span.Finish()

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
