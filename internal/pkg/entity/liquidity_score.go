package entity

import (
	"math"
)

const (
	UPPER_BOUND              = 1e12
	NORMALIZE_K_CONSTANT     = 0.95
	TVL_IN_USD_MIN_THRESHOLD = 1000
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
	// In some cases (especially limit-order), we can not generate any trade data from pools due to some abnormal reasons
	// if these pools have large tvl, we need to set level of it to 1 to make it valid in range when finding best pool ids
	if s.TvlInUsd > TVL_IN_USD_MIN_THRESHOLD && s.Level == 0 {
		s.Level = 2
	}
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

func GetMinScore(amountInUsd, threshold float64) float64 {
	if amountInUsd < threshold {
		return 0.0
	}
	score_exp := math.Floor(math.Log10(amountInUsd)) - 2
	if score_exp < 0 {
		return 0.0
	}

	return score_exp * 1e13
}
