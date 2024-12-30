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

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type LiquidityScoreIndexPoolsJob struct {
	indexUsecase               ITradeGeneratorUsecase
	updatePoolScores           IUpdatePoolScores
	blacklistIndexPoolsUsecase IBlacklistIndexPoolsUsecase
	config                     LiquidityScoreIndexPoolsJobConfig
}

func NewLiquidityScoreIndexPoolsJob(
	indexUseCase ITradeGeneratorUsecase,
	updatePoolScores IUpdatePoolScores,
	blacklistIndexPoolsUsecase IBlacklistIndexPoolsUsecase,
	config LiquidityScoreIndexPoolsJobConfig) *LiquidityScoreIndexPoolsJob {
	return &LiquidityScoreIndexPoolsJob{
		indexUsecase:               indexUseCase,
		config:                     config,
		updatePoolScores:           updatePoolScores,
		blacklistIndexPoolsUsecase: blacklistIndexPoolsUsecase,
	}
}

func (job *LiquidityScoreIndexPoolsJob) Run(ctx context.Context) {
	ticker := time.NewTicker(job.config.Interval)
	defer ticker.Stop()

	for {
		job.scanAndIndex(ctxutils.NewJobCtx(ctx))
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

func (job *LiquidityScoreIndexPoolsJob) scanAndIndex(ctx context.Context) {
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
	err := job.runScanJob(ctxutils.NewJobCtx(ctx))
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

func (job *LiquidityScoreIndexPoolsJob) runScanJob(ctx context.Context) error {
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

	go job.indexUsecase.Handle(ctx, output, totalBlacklistPools)

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
