package poolrank

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/redis/go-redis/v9"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type redisRepository struct {
	redisClient redis.UniversalClient

	keyGenerator *keyGenerator

	config RedisRepositoryConfig
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *redisRepository {
	return &redisRepository{
		redisClient:  redisClient,
		keyGenerator: NewKeyGenerator(config.Prefix),
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

	cmders, err := r.redisClient.Pipelined(
		ctx, func(tx redis.Pipeliner) error {
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.directPairKey(SortByTVL, tokenIn, tokenOut), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.DirectPoolsCount,
				},
			)

			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToWhitelistPairKey(SortByTVL), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.WhitelistPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(SortByTVL, tokenIn), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.TokenInPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(SortByTVL, tokenOut), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.TokenOutPoolCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.directPairKey(SortByAmplifiedTvl, tokenIn, tokenOut), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlDirectPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToWhitelistPairKey(SortByAmplifiedTvl), &redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlWhitelistPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTvl, tokenIn),
				&redis.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlTokenInPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTvl, tokenOut),
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
func (r *redisRepository) FindGlobalBestPools(ctx context.Context, poolCount int64) []string {
	return r.redisClient.ZRevRangeByScore(ctx, r.keyGenerator.globalSortedSetKey(SortByTVL), &redis.ZRangeBy{
		Min:   "0",
		Max:   "+inf",
		Count: poolCount,
	}).Val()
}

func (r *redisRepository) AddToSortedSetScoreByTvl(
	ctx context.Context,
	pool *entity.Pool,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
) error {
	return r.AddToSortedSet(ctx, token0, token1, isToken0Whitelisted, isToken1Whitelisted,
		SortByTVL, pool.Address, pool.ReserveUsd, true)
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

func (r *redisRepository) AddToSortedSetScoreByAmplifiedTvl(
	ctx context.Context,
	pool *entity.Pool,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
) error {
	return r.AddToSortedSet(ctx, token0, token1, isToken0Whitelisted, isToken1Whitelisted,
		SortByAmplifiedTvl, pool.Address, pool.AmplifiedTvl, false)
}
