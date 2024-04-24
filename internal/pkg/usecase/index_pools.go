package usecase

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/iter"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

type IndexPoolsUseCase struct {
	poolRepo     IPoolRepository
	poolRankRepo IPoolRankRepository

	onchainPriceRepo IOnchainPriceRepository

	config IndexPoolsConfig

	mu sync.RWMutex
}

type IndexResult int

const (
	INDEX_RESULT_SUCCESS IndexResult = iota
	INDEX_RESULT_FAIL
	INDEX_RESULT_SKIP_OLD
)

func NewIndexPoolsUseCase(
	poolRepo IPoolRepository,
	poolRankRepo IPoolRankRepository,
	onchainPriceRepo IOnchainPriceRepository,
	config IndexPoolsConfig,
) *IndexPoolsUseCase {
	return &IndexPoolsUseCase{
		poolRepo:         poolRepo,
		poolRankRepo:     poolRankRepo,
		onchainPriceRepo: onchainPriceRepo,
		config:           config,
	}
}

func (u *IndexPoolsUseCase) ApplyConfig(config IndexPoolsConfig) {
	u.mu.Lock()
	u.config = config
	u.mu.Unlock()
}

func (u *IndexPoolsUseCase) Handle(ctx context.Context, command dto.IndexPoolsCommand) *dto.IndexPoolsResult {
	var failedPoolAddresses []string
	var oldPoolCount = 0

	// process chunk by chunk
	chunks := lo.Chunk(command.PoolAddresses, u.config.ChunkSize)
	for _, poolAddresses := range chunks {
		allPools, err := u.poolRepo.FindByAddresses(ctx, poolAddresses)
		if err != nil {
			failedPoolAddresses = append(failedPoolAddresses, poolAddresses...)
		}

		// filter out pools that haven't been updated recently
		pools := lo.Filter(allPools, func(pool *entity.Pool, _ int) bool {
			return command.IgnorePoolsBeforeTimestamp <= 0 || (*pool).Timestamp >= command.IgnorePoolsBeforeTimestamp
		})
		oldPoolCount = len(allPools) - len(pools)

		var nativePriceByToken map[string]*routerEntity.OnchainPrice
		if u.config.EnableRankByNative && u.onchainPriceRepo != nil {
			// collect prices for all pools' tokens first
			nativePriceByToken, err = u.getPricesForAllTokens(ctx, pools)
			if err != nil {
				// for now still keep indexing with tvl in USD
				logger.Errorf(ctx, "error fetching pool tokens prices %v", err)
				nativePriceByToken = nil
			}
		}

		// if `u.config.NumParallel==0` (default) then will use GOMAXPROCS
		// should be set to higher since indexing wait for IO a lot
		mapper := iter.Mapper[*entity.Pool, IndexResult]{MaxGoroutines: u.config.MaxGoroutines}

		indexPoolsResults := mapper.Map(pools, func(pool **entity.Pool) IndexResult {
			return u.indexPool(ctx, *pool, nativePriceByToken)
		})

		for i, p := range pools {
			if indexPoolsResults[i] == INDEX_RESULT_FAIL {
				failedPoolAddresses = append(failedPoolAddresses, p.Address)
			}
		}
		mempool.ReserveMany(pools)
	}

	return dto.NewIndexPoolsResult(failedPoolAddresses, oldPoolCount)
}

// indexPool returns false if any errors occur and vice versa
func (u *IndexPoolsUseCase) indexPool(ctx context.Context, pool *entity.Pool, nativePriceByToken map[string]*routerEntity.OnchainPrice) IndexResult {
	if !pool.HasReserves() && !pool.HasAmplifiedTvl() {
		return INDEX_RESULT_SUCCESS
	}

	var tvlNative, amplifiedTvlNative float64
	var err error
	if nativePriceByToken != nil {
		tvlNative, err = business.CalculatePoolTVL(ctx, pool, nativePriceByToken)
		if err != nil {
			// just reset score here without returning error
			tvlNative = 0
		}

		var useTvl bool
		amplifiedTvlNative, useTvl, err = business.CalculatePoolAmplifiedTVL(ctx, pool, nativePriceByToken)
		if err != nil {
			// just reset score here without returning error
			amplifiedTvlNative = 0
		} else if useTvl {
			amplifiedTvlNative = tvlNative
		}
	}

	result := INDEX_RESULT_SUCCESS
	poolTokens := pool.Tokens
	for i := 0; i < len(poolTokens); i++ {
		tokenI := poolTokens[i]
		whiteListI := u.isWhitelistedToken(tokenI.Address)
		if !tokenI.Swappable || len(pool.Reserves)-1 < i {
			continue
		}
		for j := i + 1; j < len(poolTokens); j++ {
			tokenJ := poolTokens[j]
			if !tokenJ.Swappable || len(pool.Reserves)-1 < j {
				continue
			}
			whiteListJ := u.isWhitelistedToken(tokenJ.Address)

			if pool.HasReserve(pool.Reserves[i]) && pool.HasReserve(pool.Reserves[j]) {
				if u.savePoolToIndex(ctx, pool, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ, tvlNative, amplifiedTvlNative) == INDEX_RESULT_FAIL {
					result = INDEX_RESULT_FAIL
				}
			}
		}
	}
	// curve aave underlying
	if pool.Type == pooltypes.PoolTypes.CurveAave {
		var extra struct {
			UnderlyingTokens []string `json:"underlyingTokens"`
		}
		var err = json.Unmarshal([]byte(pool.StaticExtra), &extra)
		if err == nil {
			for i := 0; i < len(extra.UnderlyingTokens); i++ {
				for j := i + 1; j < len(extra.UnderlyingTokens); j++ {
					if len(pool.Reserves)-1 < j {
						continue
					}
					tokenI := extra.UnderlyingTokens[i]
					whiteListI := u.isWhitelistedToken(tokenI)
					tokenJ := extra.UnderlyingTokens[j]
					whiteListJ := u.isWhitelistedToken(tokenJ)

					if pool.HasReserve(pool.Reserves[i]) && pool.HasReserve(pool.Reserves[j]) {
						if u.savePoolToIndex(ctx, pool, tokenI, tokenJ, whiteListI, whiteListJ, tvlNative, amplifiedTvlNative) == INDEX_RESULT_FAIL {
							result = INDEX_RESULT_FAIL
						}
					}
				}
			}
		}
	}

	if pool.Type == pooltypes.PoolTypes.CurveMeta || pool.Type == pooltypes.PoolTypes.CurveStableMetaNg {
		// `underlyingTokens` here are metaCoin[0:numMetaCoin-1] ++ baseCoin[0:numBaseCoin]
		// we'll index for each pair of metaCoin and baseCoin
		// note that technically we can use this pool to swap between base coins, but we shouldn't, because:
		// - it cost less gas to swap base coins directly at base pool instead
		// - router might return incorrect output, because:
		//		- router find 2 paths, one through base pool and one through meta pool
		//		- router consume the 1st path and update balance for base pool accordingly
		//		- but that won't affect meta pool, because we're storing them separatedly in pool bucket
		//		- so router will incorrectly think that the 2nd path is still usable, while it's not
		// 	so rejecting base coin swap here will help us avoid that (note that we might still get another edge case:
		//		path1: basecoin1 -> basepool -> basecoin2
		// 		path2: basecoin1 -> metapool -> metacoin1 -> anotherpool -> basecoin2
		//		after consuming path1, router still assuming that path2 is usable while it's not
		//		to fix this we'll need to change the way we update balance for base & meta pool, which is much more complicated
		// 	)
		numUsableMetaCoin := len(poolTokens) - 1
		var extra struct {
			UnderlyingTokens []string `json:"underlyingTokens"`
		}
		var err = json.Unmarshal([]byte(pool.StaticExtra), &extra)
		numUnderlyingCoins := len(extra.UnderlyingTokens)
		if err == nil && numUnderlyingCoins > numUsableMetaCoin {
			for i := 0; i < numUsableMetaCoin; i++ {
				if !pool.HasReserve(pool.Reserves[i]) {
					continue
				}

				tokenI := poolTokens[i].Address
				whiteListI := u.isWhitelistedToken(tokenI)

				for j := numUsableMetaCoin; j < numUnderlyingCoins; j++ {
					tokenJ := extra.UnderlyingTokens[j]
					whiteListJ := u.isWhitelistedToken(tokenJ)

					if u.savePoolToIndex(ctx, pool, tokenI, tokenJ, whiteListI, whiteListJ, tvlNative, amplifiedTvlNative) == INDEX_RESULT_FAIL {
						result = INDEX_RESULT_FAIL
					}
				}
			}
		}
	}

	return result
}

func (u *IndexPoolsUseCase) isWhitelistedToken(tokenAddress string) bool {
	return u.config.WhitelistedTokenSet[tokenAddress]
}

type priceAndError struct {
	prices map[string]*routerEntity.OnchainPrice
	err    error
}

func (u *IndexPoolsUseCase) getPricesForAllTokens(ctx context.Context, pools []*entity.Pool) (map[string]*routerEntity.OnchainPrice, error) {

	// collect all tokens
	tokens := make([]string, 0, len(pools)*3)
	for _, p := range pools {
		if !p.HasReserves() {
			continue
		}
		for _, t := range p.Tokens {
			tokens = append(tokens, t.Address)
		}
	}

	// then get price for each chunks in parallel
	prices := make(map[string]*routerEntity.OnchainPrice, len(tokens))
	chunks := lo.Chunk(tokens, 100)

	mapper := iter.Mapper[[]string, priceAndError]{MaxGoroutines: u.config.MaxGoroutines}
	chunkResults := mapper.Map(chunks, func(chunk *[]string) priceAndError {
		chunkPrices, err := u.onchainPriceRepo.FindByAddresses(ctx, *chunk)
		if err != nil {
			return priceAndError{nil, err}
		}
		return priceAndError{chunkPrices, nil}
	})

	for _, res := range chunkResults {
		if res.err != nil {
			return nil, res.err
		}
		for token, price := range res.prices {
			prices[token] = price
		}
	}

	return prices, nil
}

func (u *IndexPoolsUseCase) savePoolToIndex(
	ctx context.Context, pool *entity.Pool,
	token0 string, token1 string,
	isToken0Whitelisted bool, isToken1Whitelisted bool,
	tvlNative, amplifiedTvlNative float64,
) IndexResult {
	result := INDEX_RESULT_SUCCESS

	if pool.HasReserves() {
		err := u.poolRankRepo.AddToSortedSetScoreByTvl(ctx, pool, token0, token1, isToken0Whitelisted, isToken1Whitelisted)

		if err != nil {
			result = INDEX_RESULT_FAIL
		}
	}

	if pool.HasAmplifiedTvl() {
		err := u.poolRankRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, token0, token1, isToken0Whitelisted, isToken1Whitelisted)

		if err != nil {
			result = INDEX_RESULT_FAIL
		}
	}

	if tvlNative > 0 {
		if err := u.poolRankRepo.AddToSortedSet(ctx, token0, token1, isToken0Whitelisted, isToken1Whitelisted,
			poolrank.SortByTVLNative, pool.Address, tvlNative, true); err != nil {
			// result = INDEX_RESULT_FAIL
			// do not mark fail here as we haven't fully switched to this yet
			logger.Debugf(ctx, "failed to add to sorted set %v", err)
		}
	}
	if amplifiedTvlNative > 0 {
		if err := u.poolRankRepo.AddToSortedSet(ctx, token0, token1, isToken0Whitelisted, isToken1Whitelisted,
			poolrank.SortByAmplifiedTVLNative, pool.Address, amplifiedTvlNative, false); err != nil {
			// result = INDEX_RESULT_FAIL
			// do not mark fail here as we haven't fully switched to this yet
			logger.Debugf(ctx, "failed to add to sorted set %v", err)
		}
	}

	return result
}
