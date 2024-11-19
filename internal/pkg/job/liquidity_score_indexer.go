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

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type LiquidityScoreIndexPoolsJob struct {
	indexUsecase     ITradeGeneratorUsecase
	updatePoolScores IUpdatePoolScores
	config           LiquidityScoreIndexPoolsJobConfig
}

func NewLiquidityScoreIndexPoolsJob(
	indexUseCase ITradeGeneratorUsecase,
	updatePoolScores IUpdatePoolScores,
	config LiquidityScoreIndexPoolsJobConfig) *LiquidityScoreIndexPoolsJob {
	return &LiquidityScoreIndexPoolsJob{
		indexUsecase:     indexUseCase,
		config:           config,
		updatePoolScores: updatePoolScores,
	}
}

func (job *LiquidityScoreIndexPoolsJob) Run(ctx context.Context) {
	ticker := time.NewTicker(job.config.Interval)
	defer ticker.Stop()
	var lastestFullScanRun time.Time
	// this set is kept during liquidity score calc job lifetime
	// black list of pools which are exhausted liquidity and causes swap errors which is gotten from Redis
	indexBlacklistWlPools := mapset.NewThreadUnsafeSet[string]()

	for {
		isFullScan := lastestFullScanRun.IsZero() || time.Since(lastestFullScanRun).Seconds() >= job.config.FullScanInterval.Seconds()
		if isFullScan {
			indexBlacklistWlPools.Clear()
		}
		job.scanAndIndex(ctxutils.NewJobCtx(ctx), isFullScan, indexBlacklistWlPools)
		if isFullScan {
			lastestFullScanRun = time.Now()
		}
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

func (job *LiquidityScoreIndexPoolsJob) scanAndIndex(ctx context.Context, isFullScan bool, indexBlacklistWlPools mapset.Set[string]) {
	startTime := time.Now()
	defer func() {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name":    LiquidityScoreIndexPools,
					"duration_ms": time.Since(startTime).Milliseconds(),
					"is_fullscan": isFullScan,
				},
			).
			Info("job done with duration")
	}()
	err := job.runScanJob(ctxutils.NewJobCtx(ctx), indexBlacklistWlPools)
	if err != nil {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name":    LiquidityScoreIndexPools,
					"error":       err,
					"is_fullscan": isFullScan,
				}).
			Error("job failed in generate trade data step")
	}

	err = job.runCalculationJob(ctx)
	if err != nil {
		logger.
			WithFields(ctx,
				logger.Fields{
					"job.name":    LiquidityScoreIndexPools,
					"error":       err,
					"is_fullscan": isFullScan,
				}).
			Error("job failed in liquidity score calculation step")
	}

	err = job.updatePoolScores.Handle(ctx)
	if err != nil {
		logger.WithFields(ctx,
			logger.Fields{
				"job.name":    LiquidityScoreIndexPools,
				"error":       err,
				"is_fullscan": isFullScan,
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

func (job *LiquidityScoreIndexPoolsJob) runScanJob(ctx context.Context, indexBlacklistWlPools mapset.Set[string]) error {
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

	go job.indexUsecase.Handle(ctx, output, indexBlacklistWlPools)

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
			indexBlacklistWlPools.Add(s)
			return false
		})
	}
	logger.Debugf(ctx, "Generate trade data successfully blacklist len %d\n", indexBlacklistWlPools.Cardinality())

	successedBuffer.Flush()
	if failedBuffer != nil {
		failedBuffer.Flush()
	}

	return nil
}
