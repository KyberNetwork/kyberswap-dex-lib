package usecase

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/core"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
)

type IndexPoolsUseCase struct {
	poolRepo  IPoolRepository
	routeRepo IIndexPoolsRouteRepository

	config IndexPoolsConfig

	mu sync.RWMutex
}

func NewIndexPoolsUseCase(
	poolRepo IPoolRepository,
	routeRepo IIndexPoolsRouteRepository,
	config IndexPoolsConfig,
) *IndexPoolsUseCase {
	return &IndexPoolsUseCase{
		poolRepo:  poolRepo,
		routeRepo: routeRepo,
		config:    config,
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

		for _, p := range pools {
			isSuccessful := u.indexPool(ctx, p)
			if !isSuccessful {
				failedPoolAddresses = append(failedPoolAddresses, p.Address)
			}
		}
	}

	return dto.NewIndexPoolsResult(failedPoolAddresses)
}

// indexPool returns false if any errors occur and vice versa
func (u *IndexPoolsUseCase) indexPool(ctx context.Context, pool entity.Pool) bool {
	if !pool.HasReserves() && !pool.HasAmplifiedTvl() {
		return true
	}

	result := true
	poolTokens := pool.Tokens
	for i := 0; i < len(poolTokens); i++ {
		tokenI := poolTokens[i]
		whiteListI := u.isWhitelistedToken(tokenI.Address)
		if !tokenI.Swappable {
			continue
		}
		for j := i + 1; j < len(poolTokens); j++ {
			tokenJ := poolTokens[j]
			if !tokenJ.Swappable {
				continue
			}
			whiteListJ := u.isWhitelistedToken(tokenJ.Address)
			key := core.GenDirectPairKey(tokenI.Address, tokenJ.Address)

			if pool.HasReserves() {
				err := u.routeRepo.AddToSortedSetScoreByReserveUsd(ctx, pool, key, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

				if err != nil {
					result = false
				}
			}

			if pool.HasAmplifiedTvl() {
				err := u.routeRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, key, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

				if err != nil {
					result = false
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
					tokenI := extra.UnderlyingTokens[i]
					whiteListI := u.isWhitelistedToken(tokenI)
					tokenJ := extra.UnderlyingTokens[j]
					whiteListJ := u.isWhitelistedToken(tokenJ)
					key := core.GenDirectPairKey(tokenI, tokenJ)

					if pool.HasReserves() {
						err := u.routeRepo.AddToSortedSetScoreByReserveUsd(ctx, pool, key, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							result = false
						}
					}

					if pool.HasAmplifiedTvl() {
						err := u.routeRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, key, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							result = false
						}
					}
				}
			}
		}
	}

	return result
}

func (u *IndexPoolsUseCase) isWhitelistedToken(tokenAddress string) bool {
	_, contained := u.config.WhitelistedTokenSet[tokenAddress]

	return contained
}
