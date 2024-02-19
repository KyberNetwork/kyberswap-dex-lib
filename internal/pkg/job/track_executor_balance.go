package job

import (
	"context"
	"time"

	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type ITrackExecutorBalanceUsecase interface {
	Handle(ctx context.Context) error
}

type TrackExecutorBalanceJob struct {
	trackExecutorBalanceUseCase ITrackExecutorBalanceUsecase
	config                      TrackExecutorBalanceConfig
}

func NewExecutorBalanceFetcherJob(
	trackExecutorBalanceUseCase ITrackExecutorBalanceUsecase,
	config TrackExecutorBalanceConfig,
) *TrackExecutorBalanceJob {
	return &TrackExecutorBalanceJob{
		trackExecutorBalanceUseCase: trackExecutorBalanceUseCase,
		config:                      config,
	}
}

func (j *TrackExecutorBalanceJob) Run(ctx context.Context) error {
	ticker := time.NewTicker(j.config.Interval)
	defer ticker.Stop()

	for {
		j.run(ctxutils.NewJobCtx(ctx))
		select {
		case <-ctx.Done():
			logger.
				WithFields(ctx,
					logger.Fields{
						"job.name": TrackExecutorBalance,
						"error":    ctx.Err(),
					}).
				Errorf("job error")
			return ctx.Err()
		case <-ticker.C:
			continue
		}
	}
}

func (j *TrackExecutorBalanceJob) run(ctx context.Context) {
	jobID := ctxutils.GetJobID(ctx)
	startTime := time.Now()
	defer func() {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":      jobID,
					"job.name":    TrackExecutorBalance,
					"duration_ms": time.Since(startTime).Milliseconds()},
			).
			Info("job duration")
	}()

	// TODO: Handle the return result.
	j.trackExecutorBalanceUseCase.Handle(ctx)
}
