package entity

import (
	"math"
)

const (
	UPPER_BOUND          = 1e12
	NORMALIZE_K_CONSTANT = 0.95
)

type PoolScore struct {
	Key            string
	LiquidityScore float64
	Level          int64
	Pool           string
	TvlInUsd       float64
}

/*
 * https://redis.io/kb/doc/1p7wk5is89/how-can-i-sort-a-leaderboard-on-multiple-fields
 * encoding scores: 1. Ranking has an upper bound 10^12, 2. Score variable and unbounded
 *
 */
func (s PoolScore) EncodeScore() float64 {
	score := float64(s.Level) * 10
	if s.LiquidityScore != 0.0 {
		score = score + 1.0
		return score*UPPER_BOUND + s.LiquidityScore
	}

	return score*UPPER_BOUND + normalizeLiquidity(s.TvlInUsd)
}

func normalizeLiquidity(liquidity float64) float64 {
	return UPPER_BOUND * liquidity / (liquidity + NORMALIZE_K_CONSTANT*UPPER_BOUND)
}

func GetMinScore(amountInUsd, threshold float64) (float64, error) {
	if amountInUsd < threshold {
		return float64(0), nil
	}

	return math.Floor(math.Log10(amountInUsd)) * 1e12, nil
}
