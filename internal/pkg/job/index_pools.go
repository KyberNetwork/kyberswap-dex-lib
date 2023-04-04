package job

import (
	"context"
	"sync"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type IndexPoolsJob struct {
	getAllPoolAddressesUseCase IGetAllPoolAddressesUseCase
	indexPoolsUseCase          IIndexPoolsUseCase

	config IndexPoolsJobConfig
	mu     sync.RWMutex
}

func NewIndexPoolsJob(
	poolUseCase IGetAllPoolAddressesUseCase,
	indexPoolsUseCase IIndexPoolsUseCase,
	config IndexPoolsJobConfig,
) *IndexPoolsJob {
	return &IndexPoolsJob{
		getAllPoolAddressesUseCase: poolUseCase,
		indexPoolsUseCase:          indexPoolsUseCase,
		config:                     config,
	}
}

func (u *IndexPoolsJob) ApplyConfig(indexPoolsJobIntervalSec uint64) {
	u.mu.Lock()
	u.config.IndexPoolsJobIntervalSec = indexPoolsJobIntervalSec
	u.mu.Unlock()
}

func (u *IndexPoolsJob) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(u.config.IndexPoolsJobIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Errorf("%v", ctx.Err())
		case <-ticker.C:
			u.run(ctx)
		}
	}
}

func (u *IndexPoolsJob) run(ctx context.Context) {
	poolAddresses, err := u.getAllPoolAddressesUseCase.Handle(ctx)
	if err != nil {
		logger.Errorf("error when getAllPoolAddresses pools, cause by %v", err)
		return
	}

	indexPoolsCmd := dto.IndexPoolsCommand{
		PoolAddresses: poolAddresses,
	}
	result := u.indexPoolsUseCase.Handle(ctx, indexPoolsCmd)
	if result != nil {
		logger.Errorf("some pools were failed to be indexed: %v", result.FailedPoolAddresses)
	}
}
