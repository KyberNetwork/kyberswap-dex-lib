package pathgenerator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/generatepath"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type redisRepository struct {
	redisClient redis.UniversalClient
	config      RedisRepositoryConfig
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient: redisClient,
		config:      config,
	}
}

func (r *redisRepository) GetPregenTokenAmounts(ctx context.Context) ([]generatepath.TokenAndAmounts, int64, error) {
	value, err := r.redisClient.Get(ctx, r.getFormattedKeyWithoutSourceHash(PregenTokenAmountsKey)).Result()
	if err != nil {
		return nil, 0, err
	}
	var data PregenTokenAmounts
	err = json.Unmarshal([]byte(value), &data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return nil, 0, err
	}
	logger.Infof("Got pregen kMeans data at: %s", data.Timestamp)
	return data.TokenAmounts, data.Timestamp, err
}

func (r *redisRepository) GetBestPaths(sourcesHash uint64, tokenIn string, tokenOut string) []*entity.MinimalPath {
	return r.get(sourcesHash, getTokenPairString(tokenIn, tokenOut))
}

func (r *redisRepository) SetBestPaths(sourcesHash uint64, tokenIn, tokenOut string, data []*entity.MinimalPath, ttl time.Duration) error {
	return r.set(sourcesHash, getTokenPairString(tokenIn, tokenOut), data, ttl)
}

func (r *redisRepository) set(sourcesHash uint64, key string, data []*entity.MinimalPath, ttl time.Duration) error {
	ctx := context.Background()
	fmtKey := r.getFormattedKey(sourcesHash, key)

	pathsString := make([]string, 0, len(data))
	for _, path := range data {
		pathsString = append(pathsString, path.Encode())
	}

	_, err := r.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		_, err := r.redisClient.Del(ctx, fmtKey).Result()
		if err != nil {
			return err
		}
		_, err = r.redisClient.SAdd(ctx, fmtKey, pathsString).Result()
		if err != nil {
			return err
		}
		_, err = r.redisClient.Expire(ctx, fmtKey, ttl).Result()
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		logger.
			WithFields(logger.Fields{"error": err, "key": fmtKey}).
			Error("set best paths failed")
		return err
	}
	return nil
}

func (r *redisRepository) get(sourcesHash uint64, key string) []*entity.MinimalPath {
	ctx := context.Background()
	fmtKey := r.getFormattedKey(sourcesHash, key)
	pathsMap, err := r.redisClient.SMembers(ctx, fmtKey).Result()

	if err != nil {
		logger.
			WithFields(logger.Fields{"error": err, "key": fmtKey}).
			Error("get best paths failed")
		return nil
	}

	paths := make([]*entity.MinimalPath, 0, len(pathsMap))

	for _, pathString := range pathsMap {
		paths = append(paths, entity.DecodeBestPath(pathString))
	}

	return paths
}

func (r *redisRepository) getFormattedKey(sourcesHash uint64, key string) string {
	return utils.Join(r.config.Prefix, entity.BestPathKey, sourcesHash, key)
}

func (r *redisRepository) getFormattedKeyWithoutSourceHash(key string) string {
	return utils.Join(r.config.Prefix, entity.BestPathKey, key)
}

func getTokenPairString(tokenIn, tokenOut string) string {
	return fmt.Sprintf("%s-%s", tokenIn, tokenOut)
}
