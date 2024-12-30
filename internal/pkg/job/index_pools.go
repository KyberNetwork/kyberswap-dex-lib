package job

import (
	"context"
	"sync"
	"time"

	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/pool-service/pkg/message"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type IndexPoolsJob struct {
	indexPoolsUseCase        IIndexPoolsUseCase
	poolEventsStreamConsumer consumer.Consumer[*message.EventMessage]

	lastScanSuccessTime int64

	config IndexPoolsJobConfig
	mu     sync.RWMutex
}

func NewIndexPoolsJob(indexPoolsUseCase IIndexPoolsUseCase,
	streamConsumer *consumer.PoolEventsStreamConsumer, config IndexPoolsJobConfig) *IndexPoolsJob {
	return &IndexPoolsJob{
		indexPoolsUseCase:        indexPoolsUseCase,
		poolEventsStreamConsumer: streamConsumer,
		config:                   config,
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

	count := 0
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
			forceScanAllPools := count%u.config.ForceScanAllEveryNth == 0
			u.scanAndIndex(ctxutils.NewJobCtx(ctx), forceScanAllPools)
			count += 1
		}
	}
}

func (u *IndexPoolsJob) scanAndIndex(ctx context.Context, forceScanAllPools bool) {
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

	indexPoolsCmd := dto.IndexPoolsCommand{
		UsePoolAddresses: false,
	}
	if !forceScanAllPools {
		indexPoolsCmd.IgnorePoolsBeforeTimestamp = u.lastScanSuccessTime
	}
	indexStartTime := time.Now().Unix()
	result := u.indexPoolsUseCase.Handle(ctx, indexPoolsCmd)

	var failedCount int
	if result != nil {
		failedCount = len(result.FailedPoolAddresses)
	}
	if successCount := result.TotalCount - failedCount; successCount > 0 {
		metrics.RecordIndexPoolsDelay(ctx, IndexPools, time.Since(startTime), true)
		metrics.CountIndexPools(ctx, IndexPools, true, successCount)
	}
	if failedCount > 0 {
		metrics.RecordIndexPoolsDelay(ctx, IndexPools, time.Since(startTime), false)
		metrics.CountIndexPools(ctx, IndexPools, false, failedCount)
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.id":       jobID,
					"job.name":     IndexPools,
					"total_count":  result.TotalCount,
					"failed_count": failedCount,
				}).
			Warn("job done")
		return
	} else {
		// only set if no pool failed, and set to start time instead of end time, in case there are pools updated in between
		u.lastScanSuccessTime = indexStartTime
	}

	logger.
		WithFields(ctx,
			logger.Fields{
				"job.id":      jobID,
				"job.name":    IndexPools,
				"total_count": result.TotalCount,
				"total_skip":  result.OldPoolCount,
				"forced":      forceScanAllPools,
			}).
		Info("job done")
}

func (u *IndexPoolsJob) RunStreamJob(ctx context.Context) {
	batcher := kutils.NewChanBatcher[*BatchedPoolAddress, *message.EventMessage](
		func() (batchRate time.Duration, batchCnt int) {
			return u.config.PoolEvent.BatchRate, u.config.PoolEvent.BatchSize
		}, u.handleStreamEvents)
	defer batcher.Close()

	for {
		u.poolEventsStreamConsumer.Consume(
			ctx,
			func(ctx context.Context, msg *message.EventMessage) error {
				return u.handleMessage(ctx, msg, batcher)
			})
		time.Sleep(u.config.PoolEvent.RetryInterval)
		logger.WithFields(ctx,
			logger.Fields{
				"job.name": consumer.PoolEvents,
			}).
			Info("job restarting")
	}
}

func (u *IndexPoolsJob) handleMessage(ctx context.Context,
	msg *message.EventMessage,
	poolCreatedBatcher *kutils.ChanBatcher[*BatchedPoolAddress, *message.EventMessage]) error {
	if msg == nil {
		return nil
	}
	switch msg.EventType {
	case message.EventPoolCreated:
		task := kutils.NewChanTask[*message.EventMessage](ctx)
		task.Resolve(msg, nil)
		poolCreatedBatcher.Batch(task)
	case message.EventPoolDeleted:
		payload := new(message.PoolDeletedPayload)
		err := json.Unmarshal([]byte(msg.Payload), payload)
		if err == nil {
			if err := u.indexPoolsUseCase.RemovePoolFromIndexes(ctx, &payload.PoolEntity); err != nil {
				logger.Errorf(ctx, "RemovePoolFromIndexes pool %s error %v", &payload.PoolEntity.Address, err)
			}
		}
	}

	return nil
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
		UsePoolAddresses: true,
		PoolAddresses:    poolAddresses,
	}
	result := u.indexPoolsUseCase.Handle(ctx, indexPoolsCmd)

	var failedCount int
	if result != nil {
		failedCount = len(result.FailedPoolAddresses)
	}
	totalCnt := len(poolAddresses)
	startTime := time.UnixMilli(msgs[0].Ret.TimeMs)
	if successCount := totalCnt - failedCount; successCount > 0 {
		metrics.RecordIndexPoolsDelay(ctx, consumer.PoolEvents, time.Since(startTime), true)
		metrics.CountIndexPools(ctx, consumer.PoolEvents, true, successCount)
	}
	if failedCount > 0 {
		metrics.RecordIndexPoolsDelay(ctx, consumer.PoolEvents, time.Since(startTime), false)
		metrics.CountIndexPools(ctx, consumer.PoolEvents, false, failedCount)
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
