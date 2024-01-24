package job

import (
	"context"
	"time"

	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type GeneratePathsJob struct {
	useCase                IGeneratePathUseCase
	excludedSourcesUseCase IGeneratePathUseCase
	config                 GenerateBestPathsJobConfig
}

func NewGenerateBestPathsJob(
	useCase IGeneratePathUseCase,
	excludedSourcesUseCase IGeneratePathUseCase,
	config GenerateBestPathsJobConfig,
) *GeneratePathsJob {
	return &GeneratePathsJob{
		useCase:                useCase,
		config:                 config,
		excludedSourcesUseCase: excludedSourcesUseCase,
	}
}

// Run will start generating many go routines
// to pre-generate the best paths for configured pairs.
// This may block the CPU and should have only run in separate instances.
func (j *GeneratePathsJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.config.Interval)
	defer ticker.Stop()

	for {
		j.run(ctxutils.NewJobCtx(ctx))

		select {
		case <-ctx.Done():
			logger.
				WithFields(
					logger.Fields{
						"job.name": GenerateBestPaths,
						"error":    ctx.Err(),
					}).
				Errorf("job error")
			return
		case <-ticker.C:
			continue
		}
	}
}

func (j *GeneratePathsJob) run(ctx context.Context) {
	jobID := ctxutils.GetJobID(ctx)
	start := time.Now()
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"job.id":      jobID,
					"job.name":    GenerateBestPaths,
					"duration_ms": time.Since(start).Milliseconds()},
			).
			Info("job duration")
	}()

	logger.WithFields(
		logger.Fields{
			"job.id":   jobID,
			"job.name": GenerateBestPaths,
		},
	).Info("job start")

	// Pregen should only find AMM dex (exclude PMM dex since those change very quickly)
	//j.useCase.Handle(ctx)
	j.excludedSourcesUseCase.Handle(ctx)
	logger.WithFields(
		logger.Fields{
			"job.id":   jobID,
			"job.name": GenerateBestPaths,
		},
	).Info("job done")
}
