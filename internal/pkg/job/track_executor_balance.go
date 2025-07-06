package job

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
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
			log.Ctx(ctx).Err(ctx.Err()).Str("job.name", TrackExecutorBalance).Msg("job error")
			return ctx.Err()
		case <-ticker.C:
			continue
		}
	}
}

func (j *TrackExecutorBalanceJob) run(ctx context.Context) {
	jobID := ctxutils.GetJobID(ctx)
	startTime := time.Now()

	err := j.trackExecutorBalanceUseCase.Handle(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).
			Str("job.id", jobID).
			Str("job.name", TrackExecutorBalance).
			Dur("duration_ms", time.Since(startTime)).
			Msg("job failed")
		return
	}

	log.Ctx(ctx).Info().
		Str("job.id", jobID).
		Str("job.name", TrackExecutorBalance).
		Dur("duration_ms", time.Since(startTime)).
		Msg("job done")
}
