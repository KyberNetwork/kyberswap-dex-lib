package indexpools

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type MeanType string

type FileHeader int

const (
	HarmonicMean          MeanType = "harmonic"
	GeometricMean         MeanType = "geometric"
	ArithmeticMean        MeanType = "arithmetic"
	WhitelistWhitelistKey          = "liquidityScoreTvl:whitelist"
)

const (
	Key FileHeader = iota
	Pool
	Harmonic
	Geometric
	Arithmetic
	Level
)

type ScoreOutput struct {
	scores           []entity.PoolScore
	isWhiteListScore bool
}

func NewUpdatePoolsScore(
	rankingRepo IPoolRankRepository,
	backupRankingRepo IPoolRankRepository,
	config *UpdateLiquidityScoreConfig) *UpdatePoolScores {
	return &UpdatePoolScores{
		rankingRepo:            rankingRepo,
		backupRankingRepo:      backupRankingRepo,
		config:                 config,
		ScoreWhitelistFileName: config.FilePath + WHITELIST_SCORE_FILENAME,
	}
}

func (u *UpdatePoolScores) ProcessScoreFiles(ctx context.Context, scoresFileNames []string) []error {
	result := make([]error, 0, 4)
	scoresChan := make(chan ScoreOutput, len(scoresFileNames))
	errorChan := make(chan error, len(scoresFileNames))

	go func(fileNames []string) {
		for _, name := range fileNames {
			err := u.readLiquidityScores(ctx, name, scoresChan)
			if err != nil {
				errorChan <- err
			}
		}
		close(scoresChan)
		close(errorChan)
	}(scoresFileNames)

	count := 0
	// Process scores and collect errors
	for output := range scoresChan {
		err := u.rankingRepo.AddScoreToSortedSets(ctx, output.scores)
		if u.config.EnableDoubleWrite && output.isWhiteListScore {
			whitelistScores := lo.Filter(output.scores, func(score entity.PoolScore, _ int) bool {
				if len(score.Key) < len(WhitelistWhitelistKey) {
					return false
				}
				return score.Key[len(score.Key)-len(WhitelistWhitelistKey):] == WhitelistWhitelistKey
			})

			if len(whitelistScores) != 0 {
				u.backupRankingRepo.AddScoreToSortedSets(ctx, whitelistScores)
			}
		}
		if err != nil {
			result = append(result, err)
		}
		count += len(output.scores)
	}

	// Collect remaining errors from the goroutine
	for err := range errorChan {
		result = append(result, err)
	}

	log.Ctx(ctx).Info().
		Str("struct", "UpdateLiquidityScore").
		Str("method", "Handle").
		Msgf("update liquidity scores total count %d", count)

	return result
}

func (u *UpdatePoolScores) readLiquidityScores(ctx context.Context, filename string, scores chan<- ScoreOutput) error {
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
	batch := make([]entity.PoolScore, 0, u.config.ChunkSize)
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
			log.Ctx(ctx).Err(err).
				Str("struct", "UpdateLiquidityScore").
				Str("method", "readLiquidityScores").
				Msgf("parse score %s is error", record[scoreHeader])
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
			scores <- ScoreOutput{
				scores:           batch,
				isWhiteListScore: filename == u.ScoreWhitelistFileName,
			}
			batch = make([]entity.PoolScore, 0, u.config.ChunkSize)
		}
	}
	if len(batch) != 0 {
		count += len(batch)
		scores <- ScoreOutput{
			scores:           batch,
			isWhiteListScore: filename == u.ScoreWhitelistFileName,
		}
	}
	log.Ctx(ctx).Info().
		Str("struct", "UpdateLiquidityScore").
		Str("method", "readLiquidityScores").
		Str("fileName", filename).
		Int("totalCount", count).
		Int("errorCount", errorCount).
		Msg("read done")

	return nil
}
