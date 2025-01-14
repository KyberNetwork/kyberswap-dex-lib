package indexpools

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
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
	Pool FileHeader = iota
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

func (u *UpdatePoolScores) Handle(ctx context.Context, inputFile string) error {
	scores, err := u.readLiquidityScores(ctx, inputFile)
	if err != nil {
		return err
	}

	if len(scores) == 0 {
		return nil
	}

	err = u.rankingRepo.AddToWhitelistSortedSet(ctx, scores, poolrank.SortByLiquidityScoreTvl, u.config.GetBestPoolsOptions.WhitelistPoolsCount)
	if err != nil {
		return err
	}

	return u.rankingRepo.AddToWhitelistSortedSet(ctx, scores, poolrank.SortByLiquidityScore, u.config.GetBestPoolsOptions.WhitelistPoolsCount)

}

func (u *UpdatePoolScores) readLiquidityScores(ctx context.Context, filename string) ([]entity.PoolScore, error) {
	result := []entity.PoolScore{}
	input, err := os.Open(filename)
	if err != nil {
		return result, err
	}
	defer input.Close()

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
	for {
		record, err := reader.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}

		if err != nil {
			return result, err
		}
		score, err := strconv.ParseFloat(record[scoreHeader], 64)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "UpdateLiquidityScore",
					"method": "readLiquidityScores",
					"error":  err,
				}).Errorf("parse score %s is error", record[scoreHeader])
			continue
		}
		level, _ := strconv.ParseInt(record[Level], 10, 8)

		result = append(result, entity.PoolScore{
			Pool:           record[Pool],
			LiquidityScore: score,
			Level:          level,
		})
	}

	return result, nil
}
