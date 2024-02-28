package job

import (
	"context"
	"sync"
	"time"

	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/pool-service/pkg/message"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type IndexPoolsJob struct {
	getAllPoolAddressesUseCase IGetAllPoolAddressesUseCase
	indexPoolsUseCase          IIndexPoolsUseCase
	poolEventsStreamConsumer   consumer.Consumer[*message.EventMessage]

	config IndexPoolsJobConfig
	mu     sync.RWMutex
}

func NewIndexPoolsJob(poolUseCase IGetAllPoolAddressesUseCase, indexPoolsUseCase IIndexPoolsUseCase,
	streamConsumer *consumer.PoolEventsStreamConsumer, config IndexPoolsJobConfig) *IndexPoolsJob {
	return &IndexPoolsJob{
		getAllPoolAddressesUseCase: poolUseCase,
		indexPoolsUseCase:          indexPoolsUseCase,
		poolEventsStreamConsumer:   streamConsumer,
		config:                     config,
	}
}

func (u *IndexPoolsJob) ApplyConfig(config IndexPoolsJobConfig) {
	u.mu.Lock()
	u.config = config
	u.mu.Unlock()
}

func (u *IndexPoolsJob) Run(ctx context.Context) {
	go u.RunScanJob(ctx)
	u.RunStreamJob(ctx)
}

func (u *IndexPoolsJob) RunScanJob(ctx context.Context) {
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
			u.scanAndIndex(ctxutils.NewJobCtx(ctx))
		}
	}
}

func (u *IndexPoolsJob) scanAndIndex(ctx context.Context) {
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
	totalCnt := len(poolAddresses)
	metrics.HistogramIndexPoolsDelay(ctx, IndexPools, time.Since(startTime), totalCnt-failedCount, totalCnt)

	if failedCount > 0 {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":       jobID,
					"job.name":     IndexPools,
					"total_count":  totalCnt,
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
				"total_count": totalCnt,
			}).
		Info("job done")
}

type BatchedPoolAddress = kutils.ChanTask[*message.EventMessage]

func (u *IndexPoolsJob) RunStreamJob(ctx context.Context) {
	batcher := kutils.NewChanBatcher[*BatchedPoolAddress, *message.EventMessage](
		func() (batchRate time.Duration, batchCnt int) {
			return u.config.PoolEvent.BatchRate, u.config.PoolEvent.BatchSize
		}, u.handleStreamEvents)
	defer batcher.Close()
	for {
		if err := u.poolEventsStreamConsumer.Consume(ctx, func(ctx context.Context, msg *message.EventMessage) error {
			if msg == nil || msg.EventType != message.EventPoolCreated {
				return nil
			}
			task := kutils.NewChanTask[*message.EventMessage](ctx)
			task.Resolve(msg, nil)
			batcher.Batch(task)
			return nil
		}); err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"job.name": consumer.PoolEvents,
					"error":    err,
				}).
				Error("job failed, waiting to restart")
		}
		time.Sleep(u.config.PoolEvent.RetryInterval)
		logger.WithFields(ctx,
			logger.Fields{
				"job.name": consumer.PoolEvents,
			}).
			Info("job restarting")
	}
}

func (u *IndexPoolsJob) handleStreamEvents(msgs []*BatchedPoolAddress) {
	if len(msgs) == 0 {
		return
	}
	ctx := ctxutils.NewJobCtx(msgs[0].Ctx())
	jobID := ctxutils.GetJobID(ctx)

	poolAddresses := lo.Map(msgs, func(item *BatchedPoolAddress, index int) string {
		return item.Ret.PoolAddress
	})

	indexPoolsCmd := dto.IndexPoolsCommand{
		PoolAddresses: poolAddresses,
	}
	result := u.indexPoolsUseCase.Handle(ctx, indexPoolsCmd)

	var failedCount int
	if result != nil {
		failedCount = len(result.FailedPoolAddresses)
	}
	totalCnt := len(poolAddresses)
	startTime := time.UnixMilli(msgs[0].Ret.TimeMs)
	metrics.HistogramIndexPoolsDelay(ctx, consumer.PoolEvents, time.Since(startTime), totalCnt-failedCount, totalCnt)

	if failedCount > 0 {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":       jobID,
					"job.name":     consumer.PoolEvents,
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
				"job.name":    consumer.PoolEvents,
				"total_count": len(poolAddresses),
			}).
		Info("job done")
}
