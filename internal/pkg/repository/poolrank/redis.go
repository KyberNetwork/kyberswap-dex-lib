package poolrank

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/redis/go-redis/v9"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type redisRepository struct {
	redisClient redis.UniversalClient

	keyGenerator *keyGenerator

	config Config
}

func NewRedisRepository(redisClient redis.UniversalClient, config Config) *redisRepository {
	return &redisRepository{
		redisClient:  redisClient,
		keyGenerator: NewKeyGenerator(config.Redis.Prefix),
		config:       config,
	}
}

// FindBestPoolIDs ...
func (r *redisRepository) FindBestPoolIDs(
	ctx context.Context,
	tokenIn, tokenOut string,
	opt valueobject.GetBestPoolsOptions,
) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.FindBestPoolIDs")
	defer span.End()

	keyTvl := SortByTVL
	if r.config.UseNativeRanking {
		keyTvl = SortByTVLNative
	}
	keyAmplifiedTvl := SortByAmplifiedTvl
	if r.config.UseNativeRanking {
		keyAmplifiedTvl = SortByAmplifiedTVLNative
	}

	cmders, err := r.redisClient.Pipelined(
		ctx, func(tx redis.Pipeliner) error {
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.directPairKey(keyTvl, tokenIn, tokenOut), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.DirectPoolsCount,
				},
			)

			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToWhitelistPairKey(keyTvl), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.WhitelistPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(keyTvl, tokenIn), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.TokenInPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(keyTvl, tokenOut), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.TokenOutPoolCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.directPairKey(keyAmplifiedTvl, tokenIn, tokenOut), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlDirectPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToWhitelistPairKey(keyAmplifiedTvl), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlWhitelistPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(keyAmplifiedTvl, tokenIn),
				&redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlTokenInPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(keyAmplifiedTvl, tokenOut),
				&redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlTokenOutPoolCount,
				},
			)

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	directPoolIdsByTvl := cmders[0].(*redis.StringSliceCmd).Val()
	whitelistToWhitelistPoolIdsByTvl := cmders[1].(*redis.StringSliceCmd).Val()
	whitelistToTokenInPoolIdsByTvl := cmders[2].(*redis.StringSliceCmd).Val()
	whitelistToTokenOutPoolIdsByTvl := cmders[3].(*redis.StringSliceCmd).Val()

	directPoolIdsByAmplifiedTvl := cmders[4].(*redis.StringSliceCmd).Val()
	whitelistToWhitelistPoolIdsByAmplifiedTvl := cmders[5].(*redis.StringSliceCmd).Val()
	whitelistToTokenInPoolIdsByAmplifiedTvl := cmders[6].(*redis.StringSliceCmd).Val()
	whitelistToTokenOutPoolIdsByAmplifiedTvl := cmders[7].(*redis.StringSliceCmd).Val()

	poolIdSet := sets.NewString()

	logger.Debugf(ctx, "best pools by TVL %v %v %v", directPoolIdsByTvl, whitelistToWhitelistPoolIdsByTvl, whitelistToTokenInPoolIdsByTvl, whitelistToTokenOutPoolIdsByTvl)
	logger.Debugf(ctx, "best pools by aTVL %v %v %v", directPoolIdsByAmplifiedTvl, whitelistToWhitelistPoolIdsByAmplifiedTvl, whitelistToTokenInPoolIdsByAmplifiedTvl, whitelistToTokenOutPoolIdsByAmplifiedTvl)

	poolIdSet.Insert(directPoolIdsByTvl...)
	poolIdSet.Insert(directPoolIdsByAmplifiedTvl...)

	poolIdSet.Insert(whitelistToWhitelistPoolIdsByTvl...)
	poolIdSet.Insert(whitelistToWhitelistPoolIdsByAmplifiedTvl...)

	poolIdSet.Insert(whitelistToTokenInPoolIdsByTvl...)
	poolIdSet.Insert(whitelistToTokenInPoolIdsByAmplifiedTvl...)

	poolIdSet.Insert(whitelistToTokenOutPoolIdsByTvl...)
	poolIdSet.Insert(whitelistToTokenOutPoolIdsByAmplifiedTvl...)

	return poolIdSet.UnsortedList(), nil
}

// FindGlobalBestPools return pools address that has the most TVL among all pairs
func (r *redisRepository) FindGlobalBestPools(ctx context.Context, poolCount int64) ([]string, error) {
	return r.redisClient.ZRevRangeByScore(ctx, r.keyGenerator.globalSortedSetKey(SortByTVL), &redis.ZRangeBy{
		Min:   "0",
		Max:   "+inf",
		Count: poolCount,
	}).Result()
}

func (r *redisRepository) AddToSortedSet(
	ctx context.Context,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
	key string, memberName string, score float64,
	useGlobal bool,
) error {
	member := redis.Z{
		Score:  score,
		Member: memberName,
	}

	_, err := r.redisClient.TxPipelined(
		ctx, func(tx redis.Pipeliner) error {
			if useGlobal {
				tx.ZAdd(ctx, r.keyGenerator.globalSortedSetKey(key), member)
			}
			tx.ZAdd(ctx, r.keyGenerator.directPairKey(key, token0, token1), member)

			if isToken0Whitelisted && isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToWhitelistPairKey(key), member)
			}

			if isToken0Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToTokenPairKey(key, token1), member)
			}

			if isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToTokenPairKey(key, token0), member)
			}

			return nil
		},
	)

	return err
}

func (r *redisRepository) RemoveFromSortedSet(
	ctx context.Context,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
	key string, memberName string, score float64,
	useGlobal bool,
) error {

	_, err := r.redisClient.TxPipelined(
		ctx, func(tx redis.Pipeliner) error {
			if useGlobal {
				tx.ZRem(ctx, r.keyGenerator.globalSortedSetKey(key), memberName)
			}
			tx.ZRem(ctx, r.keyGenerator.directPairKey(key, token0, token1), memberName)

			if isToken0Whitelisted && isToken1Whitelisted {
				tx.ZRem(ctx, r.keyGenerator.whitelistToWhitelistPairKey(key), memberName)
			}

			if isToken0Whitelisted {
				tx.ZRem(ctx, r.keyGenerator.whitelistToTokenPairKey(key, token1), memberName)
			}

			if isToken1Whitelisted {
				tx.ZRem(ctx, r.keyGenerator.whitelistToTokenPairKey(key, token0), memberName)
			}

			return nil
		},
	)

	return err
}
