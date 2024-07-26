package job

import (
	"context"
	"time"

	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type UpdateL1FeeJob struct {
	useCase  IUpdateL1FeeUseCase
	interval time.Duration
}

func NewUpdateL1FeeJob(
	useCase IUpdateL1FeeUseCase,
	interval time.Duration,
) *UpdateL1FeeJob {
	return &UpdateL1FeeJob{
		useCase:  useCase,
		interval: interval,
	}
}

func (j *UpdateL1FeeJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.
				WithFields(ctx,
					logger.Fields{
						"job.name": UpdateL1FeeParams,
						"error":    ctx.Err(),
					}).
				Errorf("job error")
			return
		case <-ticker.C:
			j.run(ctxutils.NewJobCtx(ctx))
		}
	}
}

func (j *UpdateL1FeeJob) run(ctx context.Context) {
	jobID := ctxutils.GetJobID(ctx)
	startTime := time.Now()

	err := j.useCase.Handle(ctx)
	if err != nil {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":      jobID,
					"job.name":    UpdateL1FeeParams,
					"error":       err,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}).
			Error("job failed")
		return
	}

	logger.
		WithFields(ctx,
			logger.Fields{
				"job.id":      jobID,
				"job.name":    UpdateL1FeeParams,
				"duration_ms": time.Since(startTime).Milliseconds(),
			}).
		Info("job done")
}
