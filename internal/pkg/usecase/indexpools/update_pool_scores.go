package indexpools

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type MeanType string

type FileHeader int

const (
	HarmonicMean   MeanType = "harmonic"
	GeometricMean  MeanType = "geometric"
	ArithmeticMean MeanType = "arithmetic"
)

const (
	Key FileHeader = iota
	Pool
	Harmonic
	Geometric
	Arithmetic
	Level
)

func NewUpdatePoolsScore(rankingRepo IPoolRankRepository, config UpdateLiquidityScoreConfig) *UpdatePoolScores {
	return &UpdatePoolScores{
		rankingRepo: rankingRepo,
		config:      config,
	}
}

func (u *UpdatePoolScores) ProcessScoreFiles(ctx context.Context, scoresFileNames []string) []error {
	result := make([]error, 0, 4)
	scoresChan := make(chan []entity.PoolScore, len(scoresFileNames))

	go func(fileNames []string) []error {
		for _, name := range scoresFileNames {
			err := u.readLiquidityScores(ctx, name, scoresChan)
			if err != nil {
				result = append(result, err)
			}
		}
		close(scoresChan)

		return result

	}(scoresFileNames)

	count := 0
	for scores := range scoresChan {
		err := u.rankingRepo.AddScoreToSortedSets(ctx, scores)
		if err != nil {
			result = append(result, err)
		}
		count += len(scores)
	}
	logger.WithFields(ctx,
		logger.Fields{
			"struct": "UpdateLiquidityScore",
			"method": "Handle",
		}).Errorf("update liquidity scores total count %d", count)

	return result
}

func (u *UpdatePoolScores) SavePoolScore(ctx context.Context, poolScores []entity.PoolScore) error {
	return u.rankingRepo.AddScoreToSortedSets(ctx, poolScores)
}

func (u *UpdatePoolScores) readLiquidityScores(ctx context.Context, filename string, scores chan<- []entity.PoolScore) error {
	input, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer input.Close()
	count := 0
	errorCount := 0

	reader := csv.NewReader(bufio.NewReader(input))
	reader.Comma = ','

	var scoreHeader FileHeader
	switch MeanType(u.config.MeanType) {
	case HarmonicMean:
		scoreHeader = Harmonic
	case GeometricMean:
		scoreHeader = Geometric
	case ArithmeticMean:
		scoreHeader = Arithmetic
	}

	_, _ = reader.Read()
	batch := make([]entity.PoolScore, u.config.ChunkSize)
	for {
		record, err := reader.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
		score, err := strconv.ParseFloat(record[scoreHeader], 64)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "UpdateLiquidityScore",
					"method": "readLiquidityScores",
					"error":  err,
				}).Errorf("parse score %s is error", record[scoreHeader])
			errorCount++
			continue
		}
		level, _ := strconv.ParseInt(record[Level], 10, 8)

		batch = append(batch, entity.PoolScore{
			Key:            record[Key],
			Pool:           record[Pool],
			LiquidityScore: score,
			Level:          level,
		})

		if len(batch) == u.config.ChunkSize {
			count += len(batch)
			scores <- batch
			batch = make([]entity.PoolScore, u.config.ChunkSize)
		}
	}
	if len(batch) != 0 {
		count += len(batch)
		scores <- batch
	}
	logger.WithFields(ctx,
		logger.Fields{
			"struct":     "UpdateLiquidityScore",
			"method":     "readLiquidityScores",
			"fileName":   filename,
			"totalCount": count,
			"errorCount": errorCount,
		}).Infof("read done")

	return nil
}
