package indexpools

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/iter"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
)

type MeanType string

type FileHeader int

const (
	HarmonicMean   MeanType = "harmonic"
	GeometricMean  MeanType = "geometric"
	ArithmeticMean MeanType = "arithmetic"
	WhitelistKey            = "whitelist"
	IndexName               = "liquidityScoreTvl"
)

const (
	Key FileHeader = iota
	Pool
	Harmonic
	Geometric
	Arithmetic
	Level
)

func NewUpdatePoolsScore(
	rankingRepo IPoolRankRepository,
	config *UpdateLiquidityScoreConfig) *UpdatePoolScores {
	return &UpdatePoolScores{
		rankingRepo:  rankingRepo,
		config:       config,
		keyGenerator: *poolrank.NewKeyGenerator(config.CorrelatedPairConfig.ChainName),
	}
}

func (u *UpdatePoolScores) ProcessScoreFiles(ctx context.Context, scoresFileNames []string, invalidScoreFileName string) []error {
	errors := u.saveLiquidityScores(ctx, scoresFileNames, func(ctx context.Context, scores []entity.PoolScore) error {
		return u.rankingRepo.AddScoreToSortedSets(ctx, scores)
	})

	removeErrors := u.saveLiquidityScores(ctx, []string{invalidScoreFileName}, func(ctx context.Context, scores []entity.PoolScore) error {
		return u.rankingRepo.RemoveScoreToSortedSets(ctx, scores)
	})
	if len(removeErrors) != 0 {
		log.Ctx(ctx).Error().
			Errs("errors", removeErrors).
			Str("struct", "UpdateLiquidityScore").
			Str("method", "ProcessScoreFiles error remove invalid liquidity scores")
	}

	return errors

}

func (u *UpdatePoolScores) saveLiquidityScores(ctx context.Context, scoresFileNames []string, handler func(ctx context.Context, scores []entity.PoolScore) error) []error {
	result := []error{}
	scoresChan := make(chan []entity.PoolScore, len(scoresFileNames))
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
	sortedSetPrefix := u.config.CorrelatedPairConfig.ChainName + ":" + IndexName
	correlatedPairMaxScorePools := map[string]entity.CorrelatedPairInfo{}

	// Process scores and collect errors
	for scores := range scoresChan {
		err := handler(ctx, scores)
		for _, score := range scores {
			if !strings.Contains(score.Key, WhitelistKey) {
				tokens := strings.Split(score.Key[len(sortedSetPrefix)+1:], "-")
				if !u.config.WhitelistedTokenSet[strings.ToLower(tokens[0])] &&
					!u.config.WhitelistedTokenSet[strings.ToLower(tokens[1])] &&
					score.LiquidityScore > u.config.CorrelatedPairConfig.MinLiquidityScore && score.Level >= int64(u.config.CorrelatedPairConfig.MinLiquidityScoreLevel) {
					tokenInWhitelistKey := fmt.Sprintf("%s:%s:%s", sortedSetPrefix, tokens[0], WhitelistKey)
					whitelistTokenOutKey := fmt.Sprintf("%s:%s:%s", sortedSetPrefix, WhitelistKey, tokens[1])
					whitelistTokenInKey := fmt.Sprintf("%s:%s:%s", sortedSetPrefix, WhitelistKey, tokens[0])

					cardinality := u.rankingRepo.ZCard(ctx, []string{tokenInWhitelistKey, whitelistTokenOutKey, whitelistTokenInKey})

					encodeScore := score.EncodeScore()
					/*
					 * When we can't find any route from token in to whitelist token, but we can find route from token out to whitelist token
					 * we can definitely sure that this pair of token is correlated pair
					 * this pair hase key tokenIn-*
					 */
					if cardinality[tokenInWhitelistKey] == 0 && cardinality[whitelistTokenOutKey] != 0 {
						correlatedKey := u.keyGenerator.CorrelatedPairKeyTokenIn(tokens[0])
						if poolScore, ok := correlatedPairMaxScorePools[correlatedKey]; !ok || poolScore.Score < encodeScore {
							correlatedPairMaxScorePools[correlatedKey] = entity.CorrelatedPairInfo{
								Key:   correlatedKey,
								Token: tokens[1],
								Pool:  score.Pool,
								Score: encodeScore,
							}
						}
					}

					/*
					 * When we can't find any route from whitelist to tokenOut, but we can find route from tokenIn to whitelist
					 * we can definitely sure that this pair of token is correlated pair
					 * this pair hase key *-tokenOut
					 */
					if cardinality[whitelistTokenInKey] != 0 {
						correlatedKey := u.keyGenerator.CorrelatedPairKeyTokenOut(tokens[1])
						if poolScore, ok := correlatedPairMaxScorePools[correlatedKey]; !ok || poolScore.Score < encodeScore {
							correlatedPairMaxScorePools[correlatedKey] = entity.CorrelatedPairInfo{
								Key:   correlatedKey,
								Token: tokens[0],
								Pool:  score.Pool,
								Score: encodeScore,
							}
						}
					}
				}
			}
		}
		if err != nil {
			result = append(result, err)
		}
		count += len(scores)
	}

	// Collect remaining errors from the goroutine
	for err := range errorChan {
		result = append(result, err)
	}

	u.updateCorrelatedPairs(ctx, correlatedPairMaxScorePools)

	log.Ctx(ctx).Info().
		Str("struct", "UpdateLiquidityScore").
		Str("method", "Handle").
		Msgf("update liquidity scores total count %d numOfCorrelatedPair %d", count, len(correlatedPairMaxScorePools))

	return result
}

func (u *UpdatePoolScores) readLiquidityScores(ctx context.Context, filename string, scores chan<- []entity.PoolScore) error {
	input, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func(input *os.File) {
		_ = input.Close()
	}(input)
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
			scores <- batch
			batch = make([]entity.PoolScore, 0, u.config.ChunkSize)
		}
	}
	if len(batch) != 0 {
		count += len(batch)
		scores <- batch
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

func (u *UpdatePoolScores) updateCorrelatedPairs(ctx context.Context, correlatedPairMap map[string]entity.CorrelatedPairInfo) []error {
	correlatedPairs := lo.MapToSlice(correlatedPairMap, func(_ string, value entity.CorrelatedPairInfo) entity.CorrelatedPairInfo {
		return value
	})

	chunks := lo.Chunk(correlatedPairs, 100)
	mapper := iter.Mapper[[]entity.CorrelatedPairInfo, error]{MaxGoroutines: u.config.MaxGoroutines}
	errors := mapper.Map(chunks, func(chunk *[]entity.CorrelatedPairInfo) error {
		err := u.rankingRepo.SaveCorrelatedPair(ctx, *chunk)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "UpdateLiquidityScore").
				Str("method", "updateCorrelatedPairs").
				Msg("save to redis error")
			return err
		}

		return nil
	})

	return errors

}
