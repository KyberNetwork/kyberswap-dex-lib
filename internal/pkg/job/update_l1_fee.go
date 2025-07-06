package job

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
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
			log.Ctx(ctx).Err(ctx.Err()).Str("job.name", UpdateL1FeeParams).Msg("job error")
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
		log.Ctx(ctx).Err(err).
			Str("job.id", jobID).
			Str("job.name", UpdateL1FeeParams).
			Dur("duration_ms", time.Since(startTime)).
			Msg("job failed")
		return
	}

	log.Ctx(ctx).Info().
		Str("job.id", jobID).
		Str("job.name", UpdateL1FeeParams).
		Dur("duration_ms", time.Since(startTime)).
		Msg("job done")
}
