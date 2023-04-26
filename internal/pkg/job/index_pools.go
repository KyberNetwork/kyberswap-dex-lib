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

func (u *IndexPoolsJob) ApplyConfig(config IndexPoolsJobConfig) {
	u.mu.Lock()
	u.config = config
	u.mu.Unlock()
}

func (u *IndexPoolsJob) Run(ctx context.Context) {
	ticker := time.NewTicker(u.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.
				WithFields(logger.Fields{"error": ctx.Err()}).
				Errorf("IndexPoolsJob error")
			return
		case <-ticker.C:
			u.run(ctx)
		}
	}
}

func (u *IndexPoolsJob) run(ctx context.Context) {
	startTime := time.Now()
	defer func() {
		logger.
			WithFields(logger.Fields{"duration_ms": time.Since(startTime).Milliseconds()}).
			Info("IndexPoolsJob.run done")
	}()

	poolAddresses, err := u.getAllPoolAddressesUseCase.Handle(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"error": err}).
			Error("failed to get all pool addresses")

		return
	}

	indexPoolsCmd := dto.IndexPoolsCommand{
		PoolAddresses: poolAddresses,
	}
	result := u.indexPoolsUseCase.Handle(ctx, indexPoolsCmd)
	if result != nil {
		logger.
			WithFields(logger.Fields{"failedPoolAddresses": result.FailedPoolAddresses}).
			Error("failed to index pools")
	}
}
