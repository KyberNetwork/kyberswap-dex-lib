package poolrank

import (
	"context"
	"errors"
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/logger"
	mapset "github.com/deckarep/golang-set/v2"
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

func (r *redisRepository) findBestPoolByTvl(
	ctx context.Context,
	tokenIn, tokenOut string,
	opt valueobject.GetBestPoolsOptions,
) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.FindBestPoolIDs")
	defer span.End()

	return r.findBestPoolIDsByNativeTvl(ctx, tokenIn, tokenOut, opt)
}

func (r *redisRepository) FindBestPoolIDs(
	ctx context.Context,
	tokenIn, tokenOut string,
	amountIn float64,
	opt valueobject.GetBestPoolsOptions,
	index valueobject.IndexType,
) ([]string, error) {
	if index == valueobject.NativeTvl {
		return r.findBestPoolByTvl(ctx, tokenIn, tokenOut, opt)
	} else {
		sortBy := SortByLiquidityScoreTvl
		if index == valueobject.LiquidityScore {
			sortBy = SortByLiquidityScore
		}
		return r.findBestPoolIDsByScore(
			ctx,
			tokenIn,
			tokenOut,
			amountIn,
			opt,
			sortBy,
		)
	}
}

func (r *redisRepository) findBestPoolIDsByNativeTvl(
	ctx context.Context,
	tokenIn, tokenOut string,
	opt valueobject.GetBestPoolsOptions,
) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.FindBestPoolIDsByNativeTvl")
	defer span.End()

	tvlMap := map[string]*redis.ZRangeBy{}
	tvlMap[r.keyGenerator.directPairKey(SortByTVLNative, tokenIn, tokenOut)] = r.zrangeBy(opt.DirectPoolsCount)
	tvlMap[r.keyGenerator.whitelistToWhitelistPairKey(SortByTVLNative)] = r.zrangeBy(opt.WhitelistPoolsCount)
	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByTVLNative, tokenIn)] = r.zrangeBy(opt.TokenInPoolsCount)
	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByTVLNative, tokenOut)] = r.zrangeBy(opt.TokenOutPoolCount)

	tvlMap[r.keyGenerator.directPairKey(SortByAmplifiedTVLNative, tokenIn, tokenOut)] = r.zrangeBy(opt.AmplifiedTvlDirectPoolsCount)
	tvlMap[r.keyGenerator.whitelistToWhitelistPairKey(SortByAmplifiedTVLNative)] = r.zrangeBy(opt.AmplifiedTvlWhitelistPoolsCount)
	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTVLNative, tokenIn)] = r.zrangeBy(opt.AmplifiedTvlTokenInPoolsCount)
	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTVLNative, tokenOut)] = r.zrangeBy(opt.AmplifiedTvlTokenOutPoolCount)

	return r.findBestPoolIDs(ctx, tvlMap)
}

func (r *redisRepository) zrangeBy(counter int64) *redis.ZRangeBy {
	return &redis.ZRangeBy{
		Min:   "0",
		Max:   "+inf",
		Count: counter,
	}
}

func (r *redisRepository) findBestPoolIDsByScore(
	ctx context.Context,
	tokenIn, tokenOut string,
	amountInUsd float64,
	opt valueobject.GetBestPoolsOptions,
	sortBy string,
) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.FindBestPoolIDsByNativeTvl")
	defer span.End()

	tvlMap := map[string]*redis.ZRangeBy{}
	tvlMap[r.keyGenerator.directPairKey(SortByTVLNative, tokenIn, tokenOut)] = r.zrangeBy(opt.DirectPoolsCount)
	tvlMap[r.keyGenerator.directPairKey(SortByAmplifiedTVLNative, tokenIn, tokenOut)] = r.zrangeBy(opt.AmplifiedTvlDirectPoolsCount)

	// encode amount in to find min score
	if sortBy == SortByLiquidityScoreTvl {
		score, err := entity.GetMinScore(amountInUsd, opt.AmountInThreshold)
		if err != nil {
			return nil, err
		}
		tvlMap[r.keyGenerator.whitelistToWhitelistPairKey(SortByLiquidityScoreTvl)] = &redis.ZRangeBy{
			Min:   fmt.Sprintf("%f", score),
			Max:   "+inf",
			Count: opt.WhitelistPoolsCount,
		}
	} else {
		tvlMap[r.keyGenerator.whitelistToWhitelistPairKey(SortByLiquidityScore)] = r.zrangeBy(opt.WhitelistPoolsCount)
	}

	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByTVLNative, tokenIn)] = r.zrangeBy(opt.TokenInPoolsCount)
	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByTVLNative, tokenOut)] = r.zrangeBy(opt.TokenOutPoolCount)

	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTVLNative, tokenIn)] = r.zrangeBy(opt.AmplifiedTvlTokenInPoolsCount)
	tvlMap[r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTVLNative, tokenOut)] = r.zrangeBy(opt.AmplifiedTvlTokenOutPoolCount)

	return r.findBestPoolIDs(ctx, tvlMap)
}

func (r *redisRepository) findBestPoolIDs(ctx context.Context, params map[string]*redis.ZRangeBy) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.FindBestPoolIDs")
	defer span.End()

	cmders, err := r.redisClient.Pipelined(
		ctx, func(tx redis.Pipeliner) error {
			for key, p := range params {
				tx.ZRevRangeByScore(ctx, key, p)
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	poolIdSet := sets.NewString()
	for _, cmd := range cmders {
		poolAddresses := cmd.(*redis.StringSliceCmd).Val()
		poolIdSet.Insert(poolAddresses...)
	}

	return poolIdSet.UnsortedList(), nil
}

// FindGlobalBestPools return pools address that has the most TVL among all pairs
func (r *redisRepository) FindGlobalBestPools(ctx context.Context, poolCount int64) ([]string, error) {
	return r.redisClient.ZRevRangeByScore(ctx, r.keyGenerator.globalSortedSetKey(SortByTVLNative), &redis.ZRangeBy{
		Min:   "0",
		Max:   "+inf",
		Count: poolCount,
	}).Result()
}

func (r *redisRepository) FindGlobalBestPoolsByScores(ctx context.Context, poolCount int64, sortBy string) ([]string, error) {
	whiteListSet := mapset.NewThreadUnsafeSet[string]()
	result := make([]string, 0, poolCount)
	if sortBy == SortByLiquidityScoreTvl {
		whitelist, err := r.redisClient.ZRevRangeByScore(ctx, r.keyGenerator.whitelistToWhitelistPairKey(sortBy), &redis.ZRangeBy{
			Min:   "0",
			Max:   "+inf",
			Count: poolCount,
		}).Result()
		if err != nil {
			return whitelist, err
		}
		whiteListSet.Append(whitelist...)
		// must retain the order in whitelist set returned from redis
		result = append(result, whitelist...)

	}

	if int(poolCount)-whiteListSet.Cardinality() <= 0 {
		return result, nil
	}

	globalList, err := r.FindGlobalBestPools(ctx, poolCount)

	if err != nil {
		logger.Errorf(ctx, "failed to get global set %v err: %v\n", err)
		return result, nil
	}
	for _, pool := range globalList {
		if whiteListSet.ContainsOne(pool) {
			continue
		}
		result = append(result, pool)
		if len(result) >= int(poolCount) {
			return result, nil
		}
	}

	return result, nil

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
	key string, memberName string,
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

func (r *redisRepository) RemoveAddressFromIndex(ctx context.Context, key string, pools []string) error {
	if len(pools) == 0 {
		return nil
	}
	_, err := r.redisClient.TxPipelined(
		ctx, func(tx redis.Pipeliner) error {
			// remove pools from global and whitelist for both tvl and amplifiedtvl
			tx.ZRem(ctx, r.keyGenerator.globalSortedSetKey(key), pools)
			tx.ZRem(ctx, r.keyGenerator.whitelistToWhitelistPairKey(key), pools)

			return nil
		},
	)

	return err
}

func (r *redisRepository) GetDirectIndexLength(ctx context.Context, key, token0, token1 string) (int64, error) {
	return r.redisClient.ZCard(ctx, r.keyGenerator.directPairKey(key, token0, token1)).Result()
}

func (r *redisRepository) AddToWhitelistSortedSet(ctx context.Context, scores []entity.PoolScore, sortBy string, count int64) error {
	if len(scores) == 0 {
		return errors.New("can not add empty list to whitelist sorted set")
	}
	members := []redis.Z{}
	newPoolSet := mapset.NewThreadUnsafeSet[string]()
	for _, score := range scores {
		scoreVal := score.EncodeScore(sortBy == SortByLiquidityScoreTvl)

		members = append(members, redis.Z{
			Score:  scoreVal,
			Member: score.Pool,
		})
		newPoolSet.Add(score.Pool)
	}

	return r.redisClient.ZAdd(ctx, r.keyGenerator.whitelistToWhitelistPairKey(sortBy), members...).Err()
}
