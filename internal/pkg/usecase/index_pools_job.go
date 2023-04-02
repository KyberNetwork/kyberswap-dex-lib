package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type IndexPoolsJobUseCase struct {
	indexPoolsUserCase IIndexPoolsUseCase

	poolRepo IPoolRepository

	config IndexPoolsJobConfig
	mu     sync.RWMutex
}

func NewIndexPoolsJobUseCase(
	indexPoolsUserCase IIndexPoolsUseCase,
	poolRepo IPoolRepository,
	config IndexPoolsJobConfig,
) *IndexPoolsJobUseCase {
	return &IndexPoolsJobUseCase{
		indexPoolsUserCase: indexPoolsUserCase,
		poolRepo:           poolRepo,
		config:             config,
	}
}

func (u *IndexPoolsJobUseCase) ApplyConfig(indexPoolsJobIntervalSec uint64) {
	u.mu.Lock()
	u.config.IndexPoolsJobIntervalSec = indexPoolsJobIntervalSec
	u.mu.Unlock()
}

func (u *IndexPoolsJobUseCase) Run(ctx context.Context) {
	for {
		poolAddresses, err := u.poolRepo.FindAllAddresses(ctx)
		if err != nil {
			logger.Errorf("error when findAll pools, cause by %v", err)
			continue
		}

		indexPoolsCmd := dto.IndexPoolsCommand{
			PoolAddresses: poolAddresses,
		}
		result := u.indexPoolsUserCase.Handle(ctx, indexPoolsCmd)
		if result != nil {
			logger.Errorf("some pools were failed to be indexed: %v", result.FailedPoolAddress)
		}

		time.Sleep(time.Duration(u.config.IndexPoolsJobIntervalSec) * time.Second)
	}

}
