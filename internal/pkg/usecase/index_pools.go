package usecase

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/iter"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

type IndexPoolsUseCase struct {
	poolRepo     IPoolRepository
	poolRankRepo IPoolRankRepository

	config IndexPoolsConfig

	mu sync.RWMutex
}

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

	// process chunk by chunk
	chunks := lo.Chunk(command.PoolAddresses, u.config.ChunkSize)
	for _, poolAddresses := range chunks {
		pools, err := u.poolRepo.FindByAddresses(ctx, poolAddresses)
		if err != nil {
			failedPoolAddresses = append(failedPoolAddresses, poolAddresses...)
		}

		// Map always uses at most runtime.GOMAXPROCS goroutines
		// https://pkg.go.dev/github.com/sourcegraph/conc/iter#Map
		indexPoolsResults := iter.Map(pools, func(pool **entity.Pool) bool {
			return u.indexPool(ctx, *pool)
		})

		for i, p := range pools {
			if !indexPoolsResults[i] {
				failedPoolAddresses = append(failedPoolAddresses, p.Address)
			}
		}
		mempool.ReserveMany(pools)
	}

	return dto.NewIndexPoolsResult(failedPoolAddresses)
}

// indexPool returns false if any errors occur and vice versa
func (u *IndexPoolsUseCase) indexPool(ctx context.Context, pool *entity.Pool) bool {
	if !pool.HasReserves() && !pool.HasAmplifiedTvl() {
		return true
	}

	result := true
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
						result = false
					}
				}

				if pool.HasAmplifiedTvl() {
					err := u.poolRankRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

					if err != nil {
						result = false
					}
				}
			}
		}
	}
	// curve metapool underlying
	if pool.Type == constant.PoolTypes.CurveMeta || pool.Type == constant.PoolTypes.CurveAave {
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
								result = false
							}
						}

						if pool.HasAmplifiedTvl() {
							err := u.poolRankRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, tokenI, tokenJ, whiteListI, whiteListJ)

							if err != nil {
								result = false
							}
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
