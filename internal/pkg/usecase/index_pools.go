package usecase

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/core"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type IndexPoolsUseCase struct {
	poolRepo  IPoolRepository
	routeRepo IIndexPoolsRouteRepository

	config IndexPoolsConfig
	mu     sync.RWMutex
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

func (u *IndexPoolsUseCase) ApplyConfig(whitelistedTokensByAddress map[string]bool, chunkSize uint64) {
	u.mu.Lock()
	u.config.WhitelistedTokensByAddress = whitelistedTokensByAddress
	u.config.ChunkSize = chunkSize
	u.mu.Unlock()
}

func (u *IndexPoolsUseCase) Handle(ctx context.Context, command dto.IndexPoolsCommand) *dto.IndexPoolsResult {
	var failedPoolAddresses []string
	poolAddresses := command.PoolAddresses

	// process chunk by chunk
	chunkSize := int(u.config.ChunkSize)
	start := 0
	for start < len(poolAddresses) {
		end := start + chunkSize
		if end > len(poolAddresses) {
			end = len(poolAddresses)
		}

		// TODO: update poolRepo to get pools from Pool Service API after separating Redis
		pools, err := u.poolRepo.FindByAddresses(ctx, poolAddresses[start:end])
		if err != nil {
			logger.Errorf("failed to find pools by addresses, cause by %v", err)
			failedPoolAddresses = append(failedPoolAddresses, poolAddresses[start:end]...)
		}

		for _, p := range pools {
			isSuccessful := u.indexPool(ctx, p)
			if !isSuccessful {
				failedPoolAddresses = append(failedPoolAddresses, p.Address)
			}
			logger.Infof("index pool successfully: %s", p.Address)
		}

		start += chunkSize
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
		whiteListI := u.config.WhitelistedTokensByAddress[tokenI.Address]
		if !tokenI.Swappable {
			continue
		}
		for j := i + 1; j < len(poolTokens); j++ {
			tokenJ := poolTokens[j]
			if !tokenJ.Swappable {
				continue
			}
			whiteListJ := u.config.WhitelistedTokensByAddress[tokenJ.Address]
			key := core.GenDirectPairKey(tokenI.Address, tokenJ.Address)

			if pool.HasReserves() {
				err := u.routeRepo.AddToSortedSetScoreByReserveUsd(ctx, pool, key, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

				if err != nil {
					logger.Errorf("failed to AddToSortedSetScoreByReserveUsd, err: %v", err)
					result = false
				}
			}

			if pool.HasAmplifiedTvl() {
				err := u.routeRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, key, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

				if err != nil {
					logger.Errorf("failed to AddToSortedSetScoreByReserveUsd, err: %v", err)
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
					whiteListI := u.config.WhitelistedTokensByAddress[tokenI]
					tokenJ := extra.UnderlyingTokens[j]
					whiteListJ := u.config.WhitelistedTokensByAddress[tokenJ]
					key := core.GenDirectPairKey(tokenI, tokenJ)

					if pool.HasReserves() {
						err := u.routeRepo.AddToSortedSetScoreByReserveUsd(ctx, pool, key, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							logger.Errorf("failed to AddToSortedSetScoreByReserveUsd, err: %v", err)
							result = false
						}
					}

					if pool.HasAmplifiedTvl() {
						err := u.routeRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, key, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							logger.Errorf("failed to AddToSortedSetScoreByAmplifiedTvl, err: %v", err)
							result = false
						}
					}
				}
			}
		}
	}

	return result
}
