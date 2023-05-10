package poolrank

import (
	"context"

	"github.com/redis/go-redis/v9"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
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

// FindBestPoolIDs
/*
	- **Idea**: Load only pools that could be involved in route finding. In many cases, users trade tokens with only 1 liquidity pool (especially DEXTools’s users on BSC or Polygon), it is redundant to load so many pools where we only need to load a few pools to calculate rates.
	- **Details**:
	    1. If tokenIn and tokenOut are both whitelisted tokens:
	        1. Load whitelisted pools and do calculation as current.
	    2. Else, if tokenIn is a whitelisted token:
	        1. Load (tokenOut-whitelisted_tokens) pools.
	        2. Filter wTokens = whitelisted_tokens that have at least a pool with tokenOut.
	            1. wTokens.length = 0 → no route.
	            2. wTokens.length = 1 & wTokens[0] = tokenIn → don’t need to load anything, can only trade directly from tokenIn → tokenOut.
	            3. wTokens.length > 1 → load whitelisted pools and do calculation.
	    3. Else, if tokenOut is a whitelisted token:
	        1. Do similar logic to (2)
	    4. Else (both tokens are not whitelisted tokens):
	        1. Load all pools related to (tokenIn-tokenOut), (tokenIn - whitelisted_tokens), (tokenOut - whitelisted_tokens).
	        2. Filter out wTokens = whitelisted_tokens related to either tokenIn or tokenOut.
	            1. If wTokens.length ≤ 1 ⇒ don’t need to load whitelisted pools.
	            2. Otherwise, load whitelisted pools.
	        3. Do calculation based on loaded pools.
*/
func (r *redisRepository) FindBestPoolIDs(
	ctx context.Context,
	tokenIn, tokenOut string,
	isTokenInWhitelisted, isTokenOutWhitelisted bool,
	opt types.GetBestPoolsOptions,
) ([]string, error) {
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

	poolSet := sets.NewString()
	// Merge ids into poolSet
	mergeIds := func(ids []string) {
		poolSet.Insert(ids...)
	}

	// handleOneSideWhitelistToken to handle in cases of tokenIn is whitelist and tokenOut isn't whitelist, or vice versa
	// tokenPoolIds here stands for tokenIn or tokenOut that isn't whitelist in that case.
	handleOneSideWhitelistToken := func(tokenPoolIdsByTvl, tokenPoolIdsByAmplifiedTvl []string) {
		// if tokenIn/tokenOut is a whitelisted token and the other isn't:
		tokenPoolSet := sets.NewString()
		tokenPoolSet.Insert(tokenPoolIdsByTvl...)
		tokenPoolSet.Insert(tokenPoolIdsByAmplifiedTvl...)
		tokenPoolSet.Insert(directPoolIdsByTvl...)
		tokenPoolSet.Insert(directPoolIdsByAmplifiedTvl...)

		// if doesn't exist pool to tokenOut/tokenIn
		if len(tokenPoolIdsByTvl) == 0 && len(tokenPoolIdsByAmplifiedTvl) == 0 {
			// do nothing, will not merge any pools to find route, there is no route
		} else if tokenPoolSet.Len() == 1 {
			// There is only 1 path: tokenIn -> WlToken -> tokenOut
			mergeIds(directPoolIdsByTvl)
		} else {
			// load all necessary pools
			mergeIds(tokenPoolIdsByTvl)
			mergeIds(tokenPoolIdsByAmplifiedTvl)
			mergeIds(directPoolIdsByTvl)
			mergeIds(directPoolIdsByAmplifiedTvl)
			mergeIds(whitelistToWhitelistPoolIdsByTvl)
			mergeIds(whitelistToWhitelistPoolIdsByAmplifiedTvl)
		}
	}

	// If tokenIn and tokenOut are both whitelisted tokens:
	// Load whitelisted pools and do calculation as current
	if isTokenInWhitelisted && isTokenOutWhitelisted {
		mergeIds(directPoolIdsByTvl)
		mergeIds(directPoolIdsByAmplifiedTvl)
		mergeIds(whitelistToWhitelistPoolIdsByTvl)
		mergeIds(whitelistToWhitelistPoolIdsByAmplifiedTvl)
	} else if isTokenInWhitelisted {
		handleOneSideWhitelistToken(whitelistToTokenOutPoolIdsByTvl, whitelistToTokenOutPoolIdsByAmplifiedTvl)
	} else if isTokenOutWhitelisted {
		handleOneSideWhitelistToken(whitelistToTokenInPoolIdsByTvl, whitelistToTokenInPoolIdsByAmplifiedTvl)
	} else {
		// Else (both tokens are not whitelisted tokens):
		mergeIds(directPoolIdsByAmplifiedTvl)
		mergeIds(directPoolIdsByTvl)
		mergeIds(whitelistToTokenInPoolIdsByTvl)
		mergeIds(whitelistToTokenOutPoolIdsByTvl)
		mergeIds(whitelistToTokenInPoolIdsByAmplifiedTvl)
		mergeIds(whitelistToTokenOutPoolIdsByAmplifiedTvl)

		// Check whether we should load wl pools
		uniquePoolsSet := sets.NewString()
		uniquePoolsSet.Insert(whitelistToTokenInPoolIdsByTvl...)
		uniquePoolsSet.Insert(whitelistToTokenInPoolIdsByAmplifiedTvl...)
		uniquePoolsSet.Insert(whitelistToTokenOutPoolIdsByTvl...)
		uniquePoolsSet.Insert(whitelistToTokenOutPoolIdsByAmplifiedTvl...)
		uniquePools := uniquePoolsSet.UnsortedList()
		if len(uniquePools) > 1 {
			// only load if exist more than 1 wl token to go from tokenIn -> tokenOut
			// for instance: tokenIn -> w1 -> w2 -> tokenOut
			mergeIds(whitelistToWhitelistPoolIdsByTvl)
			mergeIds(whitelistToWhitelistPoolIdsByAmplifiedTvl)
		}
	}

	poolIds := poolSet.UnsortedList()

	//totalPoolIds := sets.NewString()
	//totalPoolIds.Insert(directPoolIdsByTvl...)
	//totalPoolIds.Insert(directPoolIdsByAmplifiedTvl...)
	//totalPoolIds.Insert(whitelistToWhitelistPoolIdsByTvl...)
	//totalPoolIds.Insert(whitelistToWhitelistPoolIdsByAmplifiedTvl...)
	//totalPoolIds.Insert(whitelistToTokenInPoolIdsByTvl...)
	//totalPoolIds.Insert(whitelistToTokenInPoolIdsByAmplifiedTvl...)
	//totalPoolIds.Insert(whitelistToTokenOutPoolIdsByTvl...)
	//totalPoolIds.Insert(whitelistToTokenOutPoolIdsByAmplifiedTvl...)
	//
	//fmt.Println("Improvement 1: Reduce: ", len(totalPoolIds.List())-len(poolIds), "/", len(totalPoolIds.List()))
	return poolIds, nil
}

func (r *redisRepository) AddToSortedSetScoreByTvl(
	ctx context.Context,
	pool entity.Pool,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
) error {
	member := redis.Z{
		Score:  pool.ReserveUsd,
		Member: pool.Address,
	}

	_, err := r.redisClient.TxPipelined(
		ctx, func(tx redis.Pipeliner) error {
			tx.ZAdd(ctx, r.keyGenerator.directPairKey(SortByTVL, token0, token1), member)

			if isToken0Whitelisted && isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToWhitelistPairKey(SortByTVL), member)
			}

			if isToken0Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToTokenPairKey(SortByTVL, token1), member)
			}

			if isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToTokenPairKey(SortByTVL, token0), member)
			}

			return nil
		},
	)

	return err
}

func (r *redisRepository) AddToSortedSetScoreByAmplifiedTvl(
	ctx context.Context,
	pool entity.Pool,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
) error {
	member := redis.Z{
		Score:  pool.AmplifiedTvl,
		Member: pool.Address,
	}

	_, err := r.redisClient.TxPipelined(
		ctx, func(tx redis.Pipeliner) error {
			tx.ZAdd(ctx, r.keyGenerator.directPairKey(SortByAmplifiedTvl, token0, token1), member)

			if isToken0Whitelisted && isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToWhitelistPairKey(SortByAmplifiedTvl), member)
			}

			if isToken0Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTvl, token1), member)
			}

			if isToken1Whitelisted {
				tx.ZAdd(ctx, r.keyGenerator.whitelistToTokenPairKey(SortByAmplifiedTvl, token0), member)
			}

			return nil
		},
	)

	return err
}
