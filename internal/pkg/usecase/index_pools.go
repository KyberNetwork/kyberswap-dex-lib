package usecase

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/iter"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

type IndexPoolsUseCase struct {
	poolRepo     IPoolRepository
	poolRankRepo IPoolRankRepository

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
	config IndexPoolsConfig,
) *IndexPoolsUseCase {
	return &IndexPoolsUseCase{
		poolRepo:     poolRepo,
		poolRankRepo: poolRankRepo,
		config:       config,
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
		pools, err := u.poolRepo.FindByAddresses(ctx, poolAddresses)
		if err != nil {
			failedPoolAddresses = append(failedPoolAddresses, poolAddresses...)
		}

		// if `u.config.NumParallel==0` (default) then will use GOMAXPROCS
		// should be set to higher since indexing wait for IO a lot
		mapper := iter.Mapper[*entity.Pool, IndexResult]{MaxGoroutines: u.config.MaxGoroutines}

		indexPoolsResults := mapper.Map(pools, func(pool **entity.Pool) IndexResult {
			if command.IgnorePoolsBeforeTimestamp > 0 && (*pool).Timestamp < command.IgnorePoolsBeforeTimestamp {
				// this pool has not been updated recently, skip it
				return INDEX_RESULT_SKIP_OLD
			}
			return u.indexPool(ctx, *pool)
		})

		for i, p := range pools {
			if indexPoolsResults[i] == INDEX_RESULT_FAIL {
				failedPoolAddresses = append(failedPoolAddresses, p.Address)
			} else if indexPoolsResults[i] == INDEX_RESULT_SKIP_OLD {
				oldPoolCount += 1
			}
		}
		mempool.ReserveMany(pools)
	}

	return dto.NewIndexPoolsResult(failedPoolAddresses, oldPoolCount)
}

// indexPool returns false if any errors occur and vice versa
func (u *IndexPoolsUseCase) indexPool(ctx context.Context, pool *entity.Pool) IndexResult {
	if !pool.HasReserves() && !pool.HasAmplifiedTvl() {
		return INDEX_RESULT_SUCCESS
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
				if pool.HasReserves() {
					err := u.poolRankRepo.AddToSortedSetScoreByTvl(ctx, pool, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

					if err != nil {
						result = INDEX_RESULT_FAIL
					}
				}

				if pool.HasAmplifiedTvl() {
					err := u.poolRankRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

					if err != nil {
						result = INDEX_RESULT_FAIL
					}
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
						if pool.HasReserves() {
							err := u.poolRankRepo.AddToSortedSetScoreByTvl(ctx, pool, tokenI, tokenJ, whiteListI, whiteListJ)

							if err != nil {
								result = INDEX_RESULT_FAIL
							}
						}

						if pool.HasAmplifiedTvl() {
							err := u.poolRankRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, tokenI, tokenJ, whiteListI, whiteListJ)

							if err != nil {
								result = INDEX_RESULT_FAIL
							}
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

					if pool.HasReserves() {
						err := u.poolRankRepo.AddToSortedSetScoreByTvl(ctx, pool, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							result = INDEX_RESULT_FAIL
						}
					}

					if pool.HasAmplifiedTvl() {
						err := u.poolRankRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							result = INDEX_RESULT_FAIL
						}
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
