package job

import (
	"context"
	"sync"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
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
				WithFields(ctx,
					logger.Fields{
						"job.name": IndexPools,
						"error":    ctx.Err(),
					}).
				Errorf("job error")
			return
		case <-ticker.C:
			u.run(ctxutils.NewJobCtx(ctx))
		}
	}
}

func (u *IndexPoolsJob) run(ctx context.Context) {
	jobID := ctxutils.GetJobID(ctx)
	startTime := time.Now()
	defer func() {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":      jobID,
					"job.name":    IndexPools,
					"duration_ms": time.Since(startTime).Milliseconds()},
			).
			Info("job duration")
	}()

	poolAddresses, err := u.getAllPoolAddressesUseCase.Handle(ctx)
	if err != nil {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":   jobID,
					"job.name": IndexPools,
					"error":    err,
				}).
			Error("job failed: get all pool addresses")

		return
	}

	indexPoolsCmd := dto.IndexPoolsCommand{
		PoolAddresses: poolAddresses,
	}
	result := u.indexPoolsUseCase.Handle(ctx, indexPoolsCmd)

	var failedCount int
	if result != nil {
		failedCount = len(result.FailedPoolAddresses)
	}

	if failedCount > 0 {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":       jobID,
					"job.name":     IndexPools,
					"total_count":  len(poolAddresses),
					"failed_count": failedCount,
				}).
			Warn("job done")
		return
	}

	logger.
		WithFields(ctx,
			logger.Fields{
				"job.id":      jobID,
				"job.name":    IndexPools,
				"total_count": len(poolAddresses),
			}).
		Info("job done")
}
