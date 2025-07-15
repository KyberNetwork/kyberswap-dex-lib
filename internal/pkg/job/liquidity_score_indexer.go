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
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

const NON_FILTER_ENTROPY = 1.0

type LiquidityScoreIndexPoolsJob struct {
	indexUsecase               ITradeGeneratorUsecase
	updatePoolScores           IUpdatePoolScores
	blacklistIndexPoolsUsecase IBlacklistIndexPoolsUsecase
	removePoolUsecase          IRemovePoolIndexUseCase
	poolEventsStreamConsumer   consumer.Consumer[*message.EventMessage]
	config                     *LiquidityScoreIndexPoolsJobConfig
}

func NewLiquidityScoreIndexPoolsJob(
	indexUseCase ITradeGeneratorUsecase,
	updatePoolScores IUpdatePoolScores,
	blacklistIndexPoolsUsecase IBlacklistIndexPoolsUsecase,
	removePoolUsecase IRemovePoolIndexUseCase,
	streamConsumer consumer.Consumer[*message.EventMessage],
	config *LiquidityScoreIndexPoolsJobConfig) *LiquidityScoreIndexPoolsJob {
	return &LiquidityScoreIndexPoolsJob{
		indexUsecase:               indexUseCase,
		config:                     config,
		updatePoolScores:           updatePoolScores,
		blacklistIndexPoolsUsecase: blacklistIndexPoolsUsecase,
		poolEventsStreamConsumer:   streamConsumer,
		removePoolUsecase:          removePoolUsecase,
	}
}

func (j *LiquidityScoreIndexPoolsJob) Run(ctx context.Context) {
	go j.runScanAndIndex(ctx)
	j.subscribeEventStream(ctx)
}

func (j *LiquidityScoreIndexPoolsJob) runScanAndIndex(ctx context.Context) {
	ticker := time.NewTicker(j.config.Interval)
	defer ticker.Stop()

	for {
		j.scanAndIndex(
			ctxutils.NewJobCtx(ctx),
			mapset.NewThreadUnsafeSet[indexpools.TradesGenerationInput](),
			j.config.TargetFactorEntropy)
		select {
		case <-ctx.Done():
			log.Ctx(ctx).
				Err(ctx.Err()).
				Str("job.name", LiquidityScoreIndexPools).
				Msg("job error")
			return
		case <-ticker.C:
			continue
		}
	}
}

func (j *LiquidityScoreIndexPoolsJob) subscribeEventStream(ctx context.Context) {
	batcher := kutils.NewChanBatcher[*BatchedPoolAddress, *message.EventMessage](
		func() (batchRate time.Duration, batchCnt int) {
			return j.config.PoolEvent.BatchRate, j.config.PoolEvent.BatchSize
		}, j.handleStreamEvents)
	defer batcher.Close()

	for {
		select {
		case <-ctx.Done():
			if env.IsProductionMode() {
				time.Sleep(10 * time.Second)
			}
			log.Ctx(ctx).
				Err(ctx.Err()).
				Str("job.name", IndexPools).
				Msg("job error")
		default:
			err := j.poolEventsStreamConsumer.Consume(
				ctx,
				func(ctx context.Context, msg *message.EventMessage) error {
					return j.handleMessage(ctx, msg, batcher)
				})
			time.Sleep(j.config.PoolEvent.RetryInterval)
			log.Ctx(ctx).Info().
				Err(err).
				Str("job.name", consumer.PoolEvents).
				Msg("job restarting")
		}
	}
}

func (j *LiquidityScoreIndexPoolsJob) handleMessage(ctx context.Context,
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
		if err == nil && payload.PoolEntity.Address != "" {
			if err := j.removePoolUsecase.RemovePoolAddressFromLiqScoreIndexes(ctx,
				payload.PoolEntity.Address); err != nil {
				log.Ctx(ctx).Err(err).Str("pool", payload.PoolEntity.Address).Msg("RemovePoolFromIndexes pool failed")
			}
		}
	default:
	}
	return nil
}

func (j *LiquidityScoreIndexPoolsJob) handleStreamEvents(msgs []*BatchedPoolAddress) {
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
	j.scanAndIndex(ctx, poolAddrSet, NON_FILTER_ENTROPY)
}

func (j *LiquidityScoreIndexPoolsJob) scanAndIndex(ctx context.Context,
	poolAddresses mapset.Set[indexpools.TradesGenerationInput], entropyFactor float64) {
	startTime := time.Now()
	defer func() {
		log.Ctx(ctx).Info().
			Str("job.name", LiquidityScoreIndexPools).
			Dur("duration_ms", time.Since(startTime)).
			Msg("job done with duration")
	}()
	tradeFiles := j.runScanJob(ctxutils.NewJobCtx(ctx), poolAddresses)
	if tradeFiles.IsEmpty() {
		log.Ctx(ctx).Error().
			Str("job.name", LiquidityScoreIndexPools).
			Msg("job failed in generate trade data step len of files is 0")
	}

	scoreFiles := j.runCalculationJob(ctx, tradeFiles.ToSlice(), entropyFactor)
	if len(scoreFiles) == 0 {
		log.Ctx(ctx).Error().
			Str("job.name", LiquidityScoreIndexPools).
			Msg("job failed in liquidity score calculation step len of score files is 0")
	}

	errs := j.updatePoolScores.ProcessScoreFiles(ctx, scoreFiles)
	if len(errs) != 0 {
		log.Ctx(ctx).Error().
			Str("job.name", LiquidityScoreIndexPools).
			Errs("error", errs).
			Msg("update pools for whitelist index failed")
	}

	// remove scores file name because we have multiple scores files which can increases pod storage
	tradeFiles.Each(func(file string) bool {
		err := os.Remove(file)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("job.name", LiquidityScoreIndexPools).
				Msg("remove tradeData file with err")
		}
		return false
	})

	for _, file := range scoreFiles {
		err := os.Remove(file)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("job.name", LiquidityScoreIndexPools).
				Msg("remove scores file with err")
		}
	}
}

func (j *LiquidityScoreIndexPoolsJob) runCalculationJob(ctx context.Context, tradeDataFileNames []string,
	entropyFactor float64) []string {
	// pass a float64 as a params to python job
	scoreFileNames := make([]string, 0, len(tradeDataFileNames))
	for _, tradeFile := range tradeDataFileNames {
		factorParam := strconv.FormatFloat(NON_FILTER_ENTROPY, 'f', -1, 64)
		// only apply entropyFactor for whitelist - whitelist pools
		if tradeFile == indexpools.WHITELIST_FILENAME {
			factorParam = strconv.FormatFloat(entropyFactor, 'f', -1, 64)
		}
		scoreFileName := fmt.Sprintf("%s%s", tradeFile, "-Score")
		c := exec.Command(j.config.LiquidityScoreCalcScript, factorParam, tradeFile, scoreFileName)
		var out bytes.Buffer
		var stderr bytes.Buffer
		c.Stdout = &out
		c.Stderr = &stderr

		if err := c.Run(); err != nil {
			log.Ctx(ctx).Err(err).
				Str("job.name", LiquidityScoreIndexPools).
				Str("tradeFile", tradeFile).
				Msg("error when execute liquidity calc error")
			continue
		}

		log.Ctx(ctx).Info().
			Str("job.name", LiquidityScoreIndexPools).
			Str("tradeFile", tradeFile).
			Stringer("output", &out).
			Msg("runCalculationJob finishes")
		scoreFileNames = append(scoreFileNames, scoreFileName)
	}

	return scoreFileNames
}

func (j *LiquidityScoreIndexPoolsJob) runScanJob(ctx context.Context,
	poolAddresses mapset.Set[indexpools.TradesGenerationInput]) mapset.Set[string] {
	// get blacklist index pools from local cache
	totalBlacklistPools := j.blacklistIndexPoolsUsecase.GetBlacklistIndexPools(ctx)

	result := j.indexUsecase.Handle(ctx, totalBlacklistPools, poolAddresses)
	log.Ctx(ctx).Debug().
		Str("job.name", LiquidityScoreIndexPools).
		Int("blacklist.len", result.Blacklist.Cardinality()).
		Msg("Generate trade data successfully")

	// update blacklist to local cache
	j.blacklistIndexPoolsUsecase.AddToBlacklistIndexPools(ctx, result.Blacklist.ToSlice())
	// update zero liquidity score
	if len(result.ZeroScorePools) != 0 {
		err := j.updatePoolScores.SavePoolScore(ctx, result.ZeroScorePools)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("job.name", LiquidityScoreIndexPools).
				Msg("update zero pool score failed")
		}
		if j.config.ExportZeroScores {
			zeroScoresFile, err := os.OpenFile("zero_scores.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Ctx(ctx).Err(err).
					Str("struct", "TradeDataGenerator").
					Str("method", "writeTradeData").
					Msg("init failed buffer failed")
			} else {
				defer func() { _ = zeroScoresFile.Close() }()
				zeroScoresBuffer := bufio.NewWriter(zeroScoresFile)
				for _, score := range result.ZeroScorePools {
					jsonScore, _ := json.Marshal(score)
					_, _ = zeroScoresBuffer.WriteString(string(jsonScore))
					_, _ = zeroScoresBuffer.WriteString("\n")
				}
				_ = zeroScoresBuffer.Flush()
			}

		}
	}

	return result.OutputFileNames
}
