package job

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
)

type UpdateSuggestedGasPriceJob struct {
	useCase IUpdateSuggestedGasPriceUseCase

	config UpdateSuggestedGasPriceConfig
}

func NewUpdateSuggestedGasPriceJob(
	useCase IUpdateSuggestedGasPriceUseCase,
	config UpdateSuggestedGasPriceConfig,
) *UpdateSuggestedGasPriceJob {
	return &UpdateSuggestedGasPriceJob{
		useCase: useCase,
		config:  config,
	}
}

func (j *UpdateSuggestedGasPriceJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Err(ctx.Err()).Str("job.name", UpdateSuggestedGasPrice).Msg("job error")
			return
		case <-ticker.C:
			j.run(ctxutils.NewJobCtx(ctx))
		}
	}
}

func (j *UpdateSuggestedGasPriceJob) run(ctx context.Context) {
	jobID := ctxutils.GetJobID(ctx)
	startTime := time.Now()

	result, err := j.useCase.Handle(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).
			Str("job.id", jobID).
			Str("job.name", UpdateSuggestedGasPrice).
			Dur("duration_ms", time.Since(startTime)).
			Msg("job failed")
		return
	}

	var suggestedGasPrice string
	if result != nil && result.SuggestedGasPrice != nil {
		suggestedGasPrice = result.SuggestedGasPrice.String()
	}

	log.Ctx(ctx).Info().
		Str("job.id", jobID).
		Str("job.name", UpdateSuggestedGasPrice).
		Str("suggested_gas_price", suggestedGasPrice).
		Dur("duration_ms", time.Since(startTime)).
		Msg("job done")
}
