package job

import (
	"context"
	"time"

	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
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
			logger.
				WithFields(ctx,
					logger.Fields{
						"job.name": UpdateSuggestedGasPrice,
						"error":    ctx.Err(),
					}).
				Errorf("job error")
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
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":      jobID,
					"job.name":    UpdateSuggestedGasPrice,
					"error":       err,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}).
			Error("job failed")
		return
	}

	var suggestedGasPrice string
	if result != nil && result.SuggestedGasPrice != nil {
		suggestedGasPrice = result.SuggestedGasPrice.String()
	}

	logger.
		WithFields(ctx,
			logger.Fields{
				"job.id":              jobID,
				"job.name":            UpdateSuggestedGasPrice,
				"suggested_gas_price": suggestedGasPrice,
				"duration_ms":         time.Since(startTime).Milliseconds(),
			}).
		Info("job done")
}
