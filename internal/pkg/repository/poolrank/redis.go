package poolrank

import (
	"context"
	"errors"
	"fmt"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type redisRepository struct {
	redisClient redis.UniversalClient

	keyGenerator *KeyGenerator

	config Config
}

func NewRedisRepository(redisClient redis.UniversalClient, config Config) *redisRepository {
	return &redisRepository{
		redisClient:  redisClient,
		keyGenerator: NewKeyGenerator(config.Redis.Prefix),
		config:       config,
	}
}

func (r *redisRepository) findBestPoolByTvl(ctx context.Context, tokenIn, tokenOut string,
	opt valueobject.GetBestPoolsOptions, forcePoolsForToken map[string][]string,
) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.FindBestPoolIDs")
	defer span.End()

	return r.findBestPoolIDsByNativeTvl(ctx, tokenIn, tokenOut, opt, forcePoolsForToken)
}

func (r *redisRepository) FindBestPoolIDs(ctx context.Context, tokenIn, tokenOut string, amountIn float64,
	opt valueobject.GetBestPoolsOptions, index valueobject.IndexType, forcePoolsForToken map[string][]string,
) (poolIds []string, err error) {
	if index == valueobject.NativeTvl {
		poolIds, err = r.findBestPoolByTvl(ctx, tokenIn, tokenOut, opt, forcePoolsForToken)
	} else {
		poolIds, err = r.findBestPoolIDsByScore(
			ctx,
			tokenIn,
			tokenOut,
			amountIn,
			opt,
			forcePoolsForToken,
		)
	}
	log.Ctx(ctx).Debug().Msgf("FindBestPoolIDs|index=%s|len(poolIds)=%d", index, len(poolIds))
	return poolIds, err
}

func (r *redisRepository) findBestPoolIDsByNativeTvlRedisCommands(
	ctx context.Context,
	tokenIn, tokenOut string,
	opt valueobject.GetBestPoolsOptions,
	forcePoolsForToken map[string][]string,
) (map[string]*redis.ZRangeBy, error) {
	forcePoolsForTokenIn, forcePoolsForTokenOut := forcePoolsForToken[tokenIn], forcePoolsForToken[tokenOut]

	tvlMap := map[string]*redis.ZRangeBy{}
	if len(forcePoolsForTokenIn) == 0 && len(forcePoolsForTokenOut) == 0 {
		tvlMap[r.keyGenerator.DirectPairKey(SortByTVLNative, tokenIn, tokenOut)] = r.zrangeBy(opt.DirectPoolsCount)
		tvlMap[r.keyGenerator.DirectPairKey(SortByAmplifiedTVLNative, tokenIn,
			tokenOut)] = r.zrangeBy(opt.AmplifiedTvlDirectPoolsCount)
	}

	if opt.OnlyDirectPools {
		return tvlMap, nil
	}

	tvlMap[r.keyGenerator.WhitelistToWhitelistPairKey(SortByLiquidityScoreTvl)] = r.zrangeBy(opt.WhitelistPoolsCount)

	if len(forcePoolsForTokenIn) == 0 {
		tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByTVLNative, tokenIn)] = r.zrangeBy(opt.TokenInPoolsCount)
		tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByAmplifiedTVLNative,
			tokenIn)] = r.zrangeBy(opt.AmplifiedTvlTokenInPoolsCount)
	}
	if len(forcePoolsForTokenOut) == 0 {
		tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByTVLNative, tokenOut)] = r.zrangeBy(opt.TokenOutPoolCount)
		tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByAmplifiedTVLNative,
			tokenOut)] = r.zrangeBy(opt.AmplifiedTvlTokenOutPoolCount)
	}
	return tvlMap, nil
}

func (r *redisRepository) findBestPoolIDsByNativeTvl(
	ctx context.Context,
	tokenIn, tokenOut string,
	opt valueobject.GetBestPoolsOptions,
	forcePoolsForToken map[string][]string,
) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.findBestPoolIDsByNativeTvl")
	defer span.End()

	tvlMap, err := r.findBestPoolIDsByNativeTvlRedisCommands(ctx, tokenIn, tokenOut, opt, forcePoolsForToken)
	if err != nil {
		return nil, err
	}

	poolIDs, err := r.findBestPoolIDs(ctx, tvlMap)
	if err != nil {
		return nil, err
	}

	return append(append(poolIDs, forcePoolsForToken[tokenIn]...), forcePoolsForToken[tokenOut]...), nil
}

func (r *redisRepository) zrangeBy(counter int64) *redis.ZRangeBy {
	return &redis.ZRangeBy{
		Min:   "0",
		Max:   "+inf",
		Count: counter,
	}
}

func (r *redisRepository) findBestPoolIDsByScoreRedisCommands(ctx context.Context, tokenIn, tokenOut string, amountInUsd float64,
	opt valueobject.GetBestPoolsOptions, forcePoolsForToken map[string][]string) (map[string]*redis.ZRangeBy, error) {

	// encode amount in to find min score
	score, err := entity.GetMinScore(amountInUsd, opt.AmountInThreshold)
	if err != nil {
		return nil, err
	}

	forcePoolsForTokenIn, forcePoolsForTokenOut := forcePoolsForToken[tokenIn], forcePoolsForToken[tokenOut]

	tvlMap := map[string]*redis.ZRangeBy{}
	if len(forcePoolsForTokenIn) == 0 && len(forcePoolsForTokenOut) == 0 {
		if r.config.SetsNeededTobeIndexed[string(valueobject.DIRECT)] {
			tvlMap[r.keyGenerator.DirectPairKeyWithoutSort(SortByLiquidityScoreTvl, tokenIn, tokenOut)] = &redis.ZRangeBy{
				Min:   fmt.Sprintf("%f", score),
				Max:   "+inf",
				Count: opt.DirectPoolsCount + opt.AmplifiedTvlDirectPoolsCount,
			}
		} else {
			tvlMap[r.keyGenerator.DirectPairKey(SortByTVLNative, tokenIn, tokenOut)] = r.zrangeBy(opt.DirectPoolsCount)
			tvlMap[r.keyGenerator.DirectPairKey(SortByAmplifiedTVLNative, tokenIn,
				tokenOut)] = r.zrangeBy(opt.AmplifiedTvlDirectPoolsCount)
		}
	}

	if opt.OnlyDirectPools {
		return tvlMap, nil
	}

	tvlMap[r.keyGenerator.WhitelistToWhitelistPairKey(SortByLiquidityScoreTvl)] = &redis.ZRangeBy{
		Min:   fmt.Sprintf("%f", score),
		Max:   "+inf",
		Count: opt.WhitelistPoolsCount,
	}

	if len(forcePoolsForTokenIn) == 0 {
		if r.config.SetsNeededTobeIndexed[string(valueobject.TOKEN_WHITELIST)] {
			tvlMap[r.keyGenerator.TokenToWhitelistPairKey(SortByLiquidityScoreTvl, tokenIn)] = &redis.ZRangeBy{
				Min:   fmt.Sprintf("%f", score),
				Max:   "+inf",
				Count: opt.TokenInPoolsCount + opt.AmplifiedTvlTokenInPoolsCount,
			}
		} else {
			tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByTVLNative, tokenIn)] = r.zrangeBy(opt.TokenInPoolsCount)
			tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByAmplifiedTVLNative,
				tokenIn)] = r.zrangeBy(opt.AmplifiedTvlTokenInPoolsCount)
		}
	}
	if len(forcePoolsForTokenOut) == 0 {
		if r.config.SetsNeededTobeIndexed[string(valueobject.WHITELIST_TOKEN)] {
			tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByLiquidityScoreTvl, tokenOut)] = &redis.ZRangeBy{
				Min:   fmt.Sprintf("%f", score),
				Max:   "+inf",
				Count: opt.TokenOutPoolCount + opt.AmplifiedTvlTokenOutPoolCount,
			}
		} else {
			tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByTVLNative, tokenOut)] = r.zrangeBy(opt.TokenOutPoolCount)
			tvlMap[r.keyGenerator.WhitelistToTokenPairKey(SortByAmplifiedTVLNative,
				tokenOut)] = r.zrangeBy(opt.AmplifiedTvlTokenOutPoolCount)
		}
	}

	return tvlMap, nil
}

func (r *redisRepository) findBestPoolIDsByScore(ctx context.Context, tokenIn, tokenOut string, amountInUsd float64,
	opt valueobject.GetBestPoolsOptions, forcePoolsForToken map[string][]string) ([]string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[poolrank] redisRepository.findBestPoolIDsByScore")
	defer span.End()

	tvlMap, err := r.findBestPoolIDsByScoreRedisCommands(ctx, tokenIn, tokenOut, amountInUsd, opt, forcePoolsForToken)
	if err != nil {
		return nil, err
	}

	poolIDs, err := r.findBestPoolIDs(ctx, tvlMap)
	if err != nil {
		return nil, err
	}
	return append(append(poolIDs, forcePoolsForToken[tokenIn]...), forcePoolsForToken[tokenOut]...), nil
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
	return r.redisClient.ZRevRangeByScore(ctx, r.keyGenerator.GlobalSortedSetKey(SortByTVLNative), &redis.ZRangeBy{
		Min:   "0",
		Max:   "+inf",
		Count: poolCount,
	}).Result()
}

func (r *redisRepository) FindGlobalBestPoolsByScores(ctx context.Context, poolCount int64, sortBy string) ([]string,
	error) {
	whiteListSet := mapset.NewThreadUnsafeSet[string]()
	result := make([]string, 0, poolCount)
	whitelist, err := r.redisClient.ZRevRangeByScore(ctx, r.keyGenerator.WhitelistToWhitelistPairKey(sortBy),
		&redis.ZRangeBy{
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

	if int(poolCount)-whiteListSet.Cardinality() <= 0 {
		return result, nil
	}

	globalList, err := r.FindGlobalBestPools(ctx, poolCount)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to get global set")
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

	_, err := r.redisClient.Pipelined(
		ctx, func(tx redis.Pipeliner) error {
			if useGlobal {
				tx.ZAdd(ctx, r.keyGenerator.GlobalSortedSetKey(key), member)
			}
			tx.ZAdd(ctx, r.keyGenerator.DirectPairKey(key, token0, token1), member)

			if isToken0Whitelisted && isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.WhitelistToWhitelistPairKey(key), member)
			}

			if isToken0Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.WhitelistToTokenPairKey(key, token1), member)
			}

			if isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.WhitelistToTokenPairKey(key, token0), member)
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

	_, err := r.redisClient.Pipelined(
		ctx, func(tx redis.Pipeliner) error {
			if useGlobal {
				tx.ZRem(ctx, r.keyGenerator.GlobalSortedSetKey(key), memberName)
			}
			tx.ZRem(ctx, r.keyGenerator.DirectPairKey(key, token0, token1), memberName)

			if isToken0Whitelisted && isToken1Whitelisted {
				tx.ZRem(ctx, r.keyGenerator.WhitelistToWhitelistPairKey(key), memberName)
			}

			if isToken0Whitelisted {
				tx.ZRem(ctx, r.keyGenerator.WhitelistToTokenPairKey(key, token1), memberName)
			}

			if isToken1Whitelisted {
				tx.ZRem(ctx, r.keyGenerator.WhitelistToTokenPairKey(key, token0), memberName)
			}

			return nil
		},
	)

	return err
}

func (r *redisRepository) RemoveAddressesFromWhitelistIndex(ctx context.Context, key string, pools []string,
	removeFromGlobal bool) error {
	if len(pools) == 0 {
		return nil
	}
	// remove pools from global and whitelist for both tvl and amplifiedtvl
	if removeFromGlobal {
		_, err := r.redisClient.Pipelined(
			ctx, func(tx redis.Pipeliner) error {
				tx.ZRem(ctx, r.keyGenerator.GlobalSortedSetKey(SortByTVLNative), pools)
				tx.ZRem(ctx, r.keyGenerator.WhitelistToWhitelistPairKey(key), pools)

				return nil
			},
		)
		return err
	} else {
		return r.redisClient.ZRem(ctx, r.keyGenerator.WhitelistToWhitelistPairKey(key), pools).Err()
	}
}

func (r *redisRepository) GetDirectIndexLength(ctx context.Context, key, token0, token1 string) (int64, error) {
	return r.redisClient.ZCard(ctx, r.keyGenerator.DirectPairKey(key, token0, token1)).Result()
}

func (r *redisRepository) AddScoreToSortedSets(ctx context.Context, scores []entity.PoolScore) error {
	if len(scores) == 0 {
		return errors.New("can not add empty list to whitelist sorted set")
	}
	params := map[string][]redis.Z{}

	for _, score := range scores {
		scoreVal := score.EncodeScore()
		if params[score.Key] == nil {
			params[score.Key] = []redis.Z{}
		}

		params[score.Key] = append(params[score.Key], redis.Z{
			Score:  scoreVal,
			Member: score.Pool,
		})

	}

	_, err := r.redisClient.Pipelined(
		ctx, func(tx redis.Pipeliner) error {
			for key, members := range params {
				tx.ZAdd(ctx, key, members...)
			}

			return nil
		},
	)

	return err
}
