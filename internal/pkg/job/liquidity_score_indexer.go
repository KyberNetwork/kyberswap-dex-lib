package job

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/pool-service/pkg/message"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

const NON_FILTER_ENTROPY = 1.0

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
	go job.runScanAndIndex(ctx)
	job.subscribeEventStream(ctx)
}

func (job *LiquidityScoreIndexPoolsJob) runScanAndIndex(ctx context.Context) {
	ticker := time.NewTicker(job.config.Interval)
	defer ticker.Stop()

	for {
		job.scanAndIndex(
			ctxutils.NewJobCtx(ctx),
			mapset.NewThreadUnsafeSet[indexpools.TradesGenerationInput](),
			job.config.TargetFactorEntropy)
		select {
		case <-ctx.Done():
			logger.
				WithFields(ctx,
					logger.Fields{
						"job.name": LiquidityScoreIndexPools,
						"error":    ctx.Err(),
					}).
				Errorf("job error")
			return
		case <-ticker.C:
			continue
		}
	}
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
	case message.EventPoolCreated:
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

	poolAddrSet := mapset.NewThreadUnsafeSet[indexpools.TradesGenerationInput]()
	for _, msg := range msgs {
		payload := new(message.PoolCreatedPayload)
		err := json.Unmarshal([]byte(msg.Ret.Payload), payload)
		if err != nil {
			continue
		}
		poolAddrSet.Add(indexpools.TradesGenerationInput{
			Pool:     msg.Ret.PoolAddress,
			Exchange: payload.Exchange,
		})
	}

	ctx := ctxutils.NewJobCtx(msgs[0].Ctx())
	u.scanAndIndex(ctx, poolAddrSet, NON_FILTER_ENTROPY)
}

func (job *LiquidityScoreIndexPoolsJob) scanAndIndex(ctx context.Context,
	poolAddresses mapset.Set[indexpools.TradesGenerationInput], entropyFactor float64) {
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
	tradeFiles := job.runScanJob(ctxutils.NewJobCtx(ctx), poolAddresses)
	if tradeFiles.IsEmpty() {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name": LiquidityScoreIndexPools,
				}).
			Error("job failed in generate trade data step len of files is 0")
	}

	scoreFiles := job.runCalculationJob(ctx, tradeFiles.ToSlice(), entropyFactor)
	if len(scoreFiles) == 0 {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name": LiquidityScoreIndexPools,
				}).
			Error("job failed in liquidity score calculation step len of score files is 0")
	}

	errs := job.updatePoolScores.ProcessScoreFiles(ctx, scoreFiles)
	if len(errs) != 0 {
		logger.WithFields(ctx,
			logger.Fields{
				"job.name": LiquidityScoreIndexPools,
				"error":    errs,
			}).Errorf("update pools for whitelist index failed")
	}

	// remove scores file name because we have multiple scores files which can increases pod storage
	tradeFiles.Each(func(file string) bool {
		err := os.Remove(file)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"job.name": LiquidityScoreIndexPools,
					"error":    err,
				}).Errorf("remove tradeData file with err")
		}
		return false
	})

	for _, file := range scoreFiles {
		err := os.Remove(file)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"job.name": LiquidityScoreIndexPools,
					"error":    err,
				}).Errorf("remove scores file with err")
		}
	}
}

func (job *LiquidityScoreIndexPoolsJob) runCalculationJob(ctx context.Context, tradeDataFileNames []string, entropyFactor float64) []string {
	// pass a float64 as an params to python job
	scoreFileNames := make([]string, 0, len(tradeDataFileNames))
	for _, tradeFile := range tradeDataFileNames {
		factorParam := strconv.FormatFloat(NON_FILTER_ENTROPY, 'f', -1, 64)
		// only apply entropyFactor for whitelist - whitelist pools
		if tradeFile == indexpools.WHITELIST_FILENAME {
			factorParam = strconv.FormatFloat(entropyFactor, 'f', -1, 64)
		}
		scoreFileName := fmt.Sprintf("%s%s", "Score", tradeFile)
		c := exec.Command(job.config.LiquidityScoreCalcScript, factorParam, tradeFile, scoreFileName)
		var out bytes.Buffer
		var stderr bytes.Buffer
		c.Stdout = &out
		c.Stderr = &stderr

		if err := c.Run(); err != nil {
			logger.Errorf(ctx, "error when execute liquidity calc error %v for trade file %s, output %s", err, tradeFile, stderr.String())
			continue
		}

		logger.Infof(ctx, "[runCalculationJob - %s] Finish job with output %s", tradeFile, out.String())
		scoreFileNames = append(scoreFileNames, scoreFileName)
	}

	return scoreFileNames
}

func (job *LiquidityScoreIndexPoolsJob) runScanJob(ctx context.Context, poolAddresses mapset.Set[indexpools.TradesGenerationInput]) mapset.Set[string] {
	// get blacklist index pools from local cache
	totalBlacklistPools := job.blacklistIndexPoolsUsecase.GetBlacklistIndexPools(ctx)

	result := job.indexUsecase.Handle(ctx, totalBlacklistPools, poolAddresses)
	logger.Debugf(ctx, "Generate trade data successfully blacklist len %d\n", result.Blacklist.Cardinality())

	// update blacklist to local cache
	job.blacklistIndexPoolsUsecase.AddToBlacklistIndexPools(ctx, result.Blacklist.ToSlice())
	// update zero liquidity score
	if len(result.ZeroScorePools) != 0 {
		err := job.updatePoolScores.SavePoolScore(ctx, result.ZeroScorePools)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"job.name": LiquidityScoreIndexPools,
					"error":    err,
				}).Errorf("update zero pool score failed")
		}
		if job.config.ExportZeroScores {
			zeroScoresFile, err := os.OpenFile("zero_scores.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.WithFields(ctx,
					logger.Fields{
						"struct": "TradeDataGenerator",
						"method": "writeTradeData",
						"error":  err,
					}).Errorf("init failed buffer failed")
			} else {
				defer zeroScoresFile.Close()
				zeroScoresBuffer := bufio.NewWriter(zeroScoresFile)
				for _, score := range result.ZeroScorePools {
					jsonScore, _ := json.Marshal(score)
					zeroScoresBuffer.WriteString(string(jsonScore))
					zeroScoresBuffer.WriteString("\n")
				}
				zeroScoresBuffer.Flush()
			}

		}
	}

	return result.OutputFileNames
}
