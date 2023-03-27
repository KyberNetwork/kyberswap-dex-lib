package repository

import (
	"context"

	redisv8 "github.com/go-redis/redis/v8"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"
)

const KeyWhiteList = "whitelist"
const KeyPair = "pairs"
const KeyAmplifiedTvl = "amplifiedTvl"

type RouteRedisRepository struct {
	db *redis.Redis
}

func NewRouteRedisRepository(
	db *redis.Redis,
) *RouteRedisRepository {
	return &RouteRedisRepository{
		db: db,
	}
}

// GetBestPools
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
func (r *RouteRedisRepository) GetBestPools(ctx context.Context, directPairKey, tokenIn, tokenOut string, opt usecase.GetBestPoolsOptions, whitelistI, whitelistJ bool) (*types.BestPools, error) {
	cmders, err := r.db.Client.Pipelined(
		ctx, func(tx redisv8.Pipeliner) error {
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyPair, directPairKey), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.DirectPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyPair, KeyWhiteList), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.WhitelistPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyPair, KeyWhiteList, tokenIn), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.TokenInPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyPair, KeyWhiteList, tokenOut), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.TokenOutPoolCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyAmplifiedTvl, KeyPair, directPairKey), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlDirectPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyAmplifiedTvl, KeyPair, KeyWhiteList), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlWhitelistPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyAmplifiedTvl, KeyPair, KeyWhiteList, tokenIn),
				&redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: opt.AmplifiedTvlTokenInPoolsCount,
				},
			)
			tx.ZRevRangeByScore(
				ctx, r.db.FormatKey(KeyAmplifiedTvl, KeyPair, KeyWhiteList, tokenOut),
				&redisv8.ZRangeBy{
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

	directPoolIds := cmders[0].(*redisv8.StringSliceCmd).Val()
	whitelistPoolIds := cmders[1].(*redisv8.StringSliceCmd).Val()
	tokenInPoolIds := cmders[2].(*redisv8.StringSliceCmd).Val()
	tokenOutPoolIds := cmders[3].(*redisv8.StringSliceCmd).Val()

	directPoolIdsByAmplifiedTvl := cmders[4].(*redisv8.StringSliceCmd).Val()
	whitelistPoolIdsByAmplifiedTvl := cmders[5].(*redisv8.StringSliceCmd).Val()
	tokenInPoolIdsByAmplifiedTvl := cmders[6].(*redisv8.StringSliceCmd).Val()
	tokenOutPoolIdsByAmplifiedTvl := cmders[7].(*redisv8.StringSliceCmd).Val()

	poolSet := sets.NewString()
	// Merge ids into poolSet
	mergeIds := func(ids []string) {
		poolSet.Insert(ids...)
	}

	// handleOneSideWhitelistToken to handle in cases of tokenIn is whitelist and tokenOut isn't whitelist, or vice versa
	// tokenPoolIds here stands for tokenIn or tokenOut that isn't whitelist in that case.
	handleOneSideWhitelistToken := func(tokenPoolIds, tokenPoolIdsByAmplifiedTvl []string) {
		// if tokenIn/tokenOut is a whitelisted token and the other isn't:
		tokenPoolSet := sets.NewString()
		tokenPoolSet.Insert(tokenPoolIds...)
		tokenPoolSet.Insert(tokenPoolIdsByAmplifiedTvl...)
		tokenPoolSet.Insert(directPoolIds...)
		tokenPoolSet.Insert(directPoolIdsByAmplifiedTvl...)
		uniqueTokenPools := tokenPoolSet.List()

		// if doesn't exist pool to tokenOut/tokenIn
		if len(tokenPoolIds) == 0 && len(tokenPoolIdsByAmplifiedTvl) == 0 {
			// do nothing, will not merge any pools to find route, there is no route
		} else if len(uniqueTokenPools) == 1 {
			// There is only 1 path: tokenIn -> WlToken -> tokenOut
			mergeIds(directPoolIds)
		} else {
			// load all necessary pools
			mergeIds(tokenPoolIds)
			mergeIds(tokenPoolIdsByAmplifiedTvl)
			mergeIds(directPoolIds)
			mergeIds(directPoolIdsByAmplifiedTvl)
			mergeIds(whitelistPoolIds)
			mergeIds(whitelistPoolIdsByAmplifiedTvl)
		}
	}

	// If tokenIn and tokenOut are both whitelisted tokens:
	// Load whitelisted pools and do calculation as current
	if whitelistI && whitelistJ {
		mergeIds(whitelistPoolIds)
		mergeIds(whitelistPoolIdsByAmplifiedTvl)
	} else if whitelistI {
		handleOneSideWhitelistToken(tokenOutPoolIds, tokenOutPoolIdsByAmplifiedTvl)
	} else if whitelistJ {
		handleOneSideWhitelistToken(tokenInPoolIds, tokenInPoolIdsByAmplifiedTvl)
	} else {
		// Else (both tokens are not whitelisted tokens):
		mergeIds(directPoolIdsByAmplifiedTvl)
		mergeIds(directPoolIds)
		mergeIds(tokenInPoolIds)
		mergeIds(tokenOutPoolIds)
		mergeIds(tokenInPoolIdsByAmplifiedTvl)
		mergeIds(tokenOutPoolIdsByAmplifiedTvl)

		// Check whether we should load wl pools
		uniquePoolsSet := sets.NewString()
		uniquePoolsSet.Insert(tokenInPoolIds...)
		uniquePoolsSet.Insert(tokenInPoolIdsByAmplifiedTvl...)
		uniquePoolsSet.Insert(tokenOutPoolIds...)
		uniquePoolsSet.Insert(tokenOutPoolIdsByAmplifiedTvl...)
		uniquePools := uniquePoolsSet.List()
		if len(uniquePools) > 1 {
			// only load if exist more than 1 wl token to go from tokenIn -> tokenOut
			// for instance: tokenIn -> w1 -> w2 -> tokenOut
			mergeIds(whitelistPoolIds)
			mergeIds(whitelistPoolIdsByAmplifiedTvl)
		}
	}

	poolIds := poolSet.List()

	//totalPoolIds := sets.NewString()
	//totalPoolIds.Insert(directPoolIds...)
	//totalPoolIds.Insert(directPoolIdsByAmplifiedTvl...)
	//totalPoolIds.Insert(whitelistPoolIds...)
	//totalPoolIds.Insert(whitelistPoolIdsByAmplifiedTvl...)
	//totalPoolIds.Insert(tokenInPoolIds...)
	//totalPoolIds.Insert(tokenInPoolIdsByAmplifiedTvl...)
	//totalPoolIds.Insert(tokenOutPoolIds...)
	//totalPoolIds.Insert(tokenOutPoolIdsByAmplifiedTvl...)
	//
	//fmt.Println("Improvement 1: Reduce: ", len(totalPoolIds.List())-len(poolIds), "/", len(totalPoolIds.List()))
	return &types.BestPools{
		//PoolIds: totalPoolIds.List(),
		PoolIds:          poolIds,
		WhitelistPoolIds: whitelistPoolIds,
		TokenInPoolIds:   tokenInPoolIds,
		TokenOutPoolIds:  tokenOutPoolIds,
	}, nil
}

func (r *RouteRedisRepository) AddToSortedSetScoreByReserveUsd(ctx context.Context, pool entity.Pool, key string, tokenIAddress, tokenJAddress string, whiteListI, whiteListJ bool) error {
	member := &redisv8.Z{
		Score:  pool.ReserveUsd,
		Member: pool.Address,
	}

	_, err := r.db.Client.TxPipelined(
		ctx, func(tx redisv8.Pipeliner) error {
			tx.ZAdd(ctx, r.db.FormatKey(KeyPair, key), member)
			if whiteListI && whiteListJ {
				tx.ZAdd(ctx, r.db.FormatKey(KeyPair, KeyWhiteList), member)
			}
			if whiteListI {
				tx.ZAdd(ctx, r.db.FormatKey(KeyPair, KeyWhiteList, tokenJAddress), member)
			}
			if whiteListJ {
				tx.ZAdd(ctx, r.db.FormatKey(KeyPair, KeyWhiteList, tokenIAddress), member)
			}

			return nil
		},
	)

	return err
}

func (r *RouteRedisRepository) AddToSortedSetScoreByAmplifiedTvl(ctx context.Context, pool entity.Pool, key string, tokenIAddress, tokenJAddress string, whiteListI, whiteListJ bool) error {
	member := &redisv8.Z{
		Score:  pool.AmplifiedTvl,
		Member: pool.Address,
	}

	_, err := r.db.Client.TxPipelined(
		ctx, func(tx redisv8.Pipeliner) error {
			tx.ZAdd(ctx, r.db.FormatKey(KeyAmplifiedTvl, KeyPair, key), member)
			if whiteListI && whiteListJ {
				tx.ZAdd(ctx, r.db.FormatKey(KeyAmplifiedTvl, KeyPair, KeyWhiteList), member)
			}
			if whiteListI {
				tx.ZAdd(
					ctx,
					r.db.FormatKey(KeyAmplifiedTvl, KeyPair, KeyWhiteList, tokenJAddress),
					member,
				)
			}
			if whiteListJ {
				tx.ZAdd(
					ctx,
					r.db.FormatKey(KeyAmplifiedTvl, KeyPair, KeyWhiteList, tokenIAddress),
					member,
				)
			}
			return nil
		},
	)

	return err
}
