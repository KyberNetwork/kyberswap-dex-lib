package job

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/pool-service/pkg/message"
	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/util/env"
	mapset "github.com/deckarep/golang-set/v2"
)

type LiquidityScoreIndexPoolsJob struct {
	indexUsecase               ITradeGeneratorUsecase
	updatePoolScores           IUpdatePoolScores
	blacklistIndexPoolsUsecase IBlacklistIndexPoolsUsecase
	removePoolUsecase          IRemovePoolIndexUseCase
	poolEventsStreamConsumer   consumer.Consumer[*message.EventMessage]
	config                     LiquidityScoreIndexPoolsJobConfig
}

func NewLiquidityScoreIndexPoolsJob(
	indexUseCase ITradeGeneratorUsecase,
	updatePoolScores IUpdatePoolScores,
	blacklistIndexPoolsUsecase IBlacklistIndexPoolsUsecase,
	removePoolUsecase IRemovePoolIndexUseCase,
	streamConsumer consumer.Consumer[*message.EventMessage],
	config LiquidityScoreIndexPoolsJobConfig) *LiquidityScoreIndexPoolsJob {
	return &LiquidityScoreIndexPoolsJob{
		indexUsecase:               indexUseCase,
		config:                     config,
		updatePoolScores:           updatePoolScores,
		blacklistIndexPoolsUsecase: blacklistIndexPoolsUsecase,
		poolEventsStreamConsumer:   streamConsumer,
		removePoolUsecase:          removePoolUsecase,
	}
}

func (job *LiquidityScoreIndexPoolsJob) Run(ctx context.Context) {
	job.subscribeEventStream(ctx)
}

func (u *LiquidityScoreIndexPoolsJob) subscribeEventStream(ctx context.Context) {
	batcher := kutils.NewChanBatcher[*BatchedPoolAddress, *message.EventMessage](
		func() (batchRate time.Duration, batchCnt int) {
			return u.config.PoolEvent.BatchRate, u.config.PoolEvent.BatchSize
		}, u.handleStreamEvents)
	defer batcher.Close()

	for {
		select {
		case <-ctx.Done():
			if env.IsProductionMode(u.config.Env) {
				time.Sleep(10 * time.Second)
			}
			logger.
				WithFields(ctx,
					logger.Fields{
						"job.name": IndexPools,
						"error":    ctx.Err(),
					}).
				Errorf("job error")
		default:
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
}

func (u *LiquidityScoreIndexPoolsJob) handleMessage(ctx context.Context,
	msg *message.EventMessage,
	poolCreatedBatcher *kutils.ChanBatcher[*BatchedPoolAddress, *message.EventMessage]) error {
	if msg == nil {
		return nil
	}
	switch msg.EventType {
	// keep message.EventUnspecified to backward compatible with the old events in redis stream, will be removed later
	case message.EventPoolCreated, message.EventPoolUpdated, message.EventUnspecified:
		task := kutils.NewChanTask[*message.EventMessage](ctx)
		task.Resolve(msg, nil)
		poolCreatedBatcher.Batch(task)
	case message.EventPoolDeleted:
		payload := new(message.PoolDeletedPayload)
		err := json.Unmarshal([]byte(msg.Payload), payload)
		if err == nil {
			if payload.PoolEntity.Address != "" {
				err := u.removePoolUsecase.RemovePoolAddressFromLiqScoreIndexes(ctx, payload.PoolEntity.Address)
				if err != nil {
					logger.Errorf(ctx, "RemovePoolFromIndexes pool %s error %v", &payload.PoolEntity.Address, err)
				}
			}
		}
	}

	return nil
}

func (u *LiquidityScoreIndexPoolsJob) handleStreamEvents(msgs []*BatchedPoolAddress) {
	if len(msgs) == 0 {
		return
	}

	poolAddrSet := mapset.NewThreadUnsafeSet[string]()
	for _, msg := range msgs {
		poolAddrSet.Add(msg.Ret.PoolAddress)
	}

	ctx := ctxutils.NewJobCtx(msgs[0].Ctx())
	u.scanAndIndex(ctx, poolAddrSet)
}

func (job *LiquidityScoreIndexPoolsJob) scanAndIndex(ctx context.Context, poolAddresses mapset.Set[string]) {
	startTime := time.Now()
	defer func() {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name":    LiquidityScoreIndexPools,
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			).
			Info("job done with duration")
	}()
	err := job.runScanJob(ctxutils.NewJobCtx(ctx), poolAddresses)
	if err != nil {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name": LiquidityScoreIndexPools,
					"error":    err,
				}).
			Error("job failed in generate trade data step")
	}

	err = job.runCalculationJob(ctx)
	if err != nil {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name": LiquidityScoreIndexPools,
					"error":    err,
				}).
			Error("job failed in liquidity score calculation step")
	}

	err = job.updatePoolScores.Handle(ctx)
	if err != nil {
		logger.WithFields(ctx,
			logger.Fields{
				"job.name": LiquidityScoreIndexPools,
				"error":    err,
			}).Errorf("update pools for whitelist index failed")
	}
}

func (job *LiquidityScoreIndexPoolsJob) runCalculationJob(ctx context.Context) error {
	c := exec.Command(job.config.LiquidityScoreCalcScript)
	var out bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &out
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("error when execute liquidity calc error %v, output %s", err, stderr.String())
	}

	logger.Infof(ctx, "[runCalculationJob] Finish job with output %s", out.String())

	return nil
}

func (job *LiquidityScoreIndexPoolsJob) runScanJob(ctx context.Context, poolAddresses mapset.Set[string]) error {
	// get blacklist index pools from local cache
	totalBlacklistPools := job.blacklistIndexPoolsUsecase.GetBlacklistIndexPools(ctx)
	newBlacklistPools := []string{}

	// open file in order to write the output
	file, err := os.Create(job.config.SuccessedFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	var failedBuffer *bufio.Writer
	if job.config.ExportFailedTrade {
		failedFile, err := os.Create(job.config.FailedFileName)
		if err != nil {
			panic(err)
		}
		defer failedFile.Close()
		failedBuffer = bufio.NewWriter(failedFile)
	}

	successedBuffer := bufio.NewWriter(file)

	// failedBuffer := bufio.NewWriter(failedFile)
	output := make(chan indexpools.TradesGenerationOutput, job.config.BatchSize)

	go job.indexUsecase.Handle(ctx, output, totalBlacklistPools, poolAddresses)

	for output := range output {
		for p, trades := range output.Successed {
			for pair, values := range trades {
				jsonStr, err := json.Marshal(values)
				if err != nil {
					continue
				}
				logger.Debugf(ctx, "Generate trade data success data %s\n", fmt.Sprintf("%s:%s:%s\n", p, pair, jsonStr))
				successedBuffer.Write([]byte(fmt.Sprintf("%s:%s:%s\n", p, pair, jsonStr)))
			}
		}

		for p, errTrades := range output.Failed {
			for _, values := range errTrades {
				jsonErr, err := json.Marshal(values)
				if err != nil {
					continue
				}
				// push logs to grafana
				if job.config.ExportFailedTrade {
					failedBuffer.Write([]byte(fmt.Sprintf("%s:%s\n", p, jsonErr)))
				} else {
					logger.Errorf(ctx, "Generate trade data failed %s:%s", p, jsonErr)
				}
			}
		}

		// update blacklist pools
		output.Blacklist.Each(func(s string) bool {
			newBlacklistPools = append(newBlacklistPools, s)
			return false
		})
	}
	logger.Debugf(ctx, "Generate trade data successfully blacklist len %d\n", len(newBlacklistPools))

	// update blacklist to local cache
	job.blacklistIndexPoolsUsecase.AddToBlacklistIndexPools(ctx, newBlacklistPools)

	successedBuffer.Flush()
	if failedBuffer != nil {
		failedBuffer.Flush()
	}

	return nil
}
