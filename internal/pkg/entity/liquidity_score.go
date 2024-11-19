package entity

import (
	"math"
)

type PoolScore struct {
	LiquidityScore float64
	Level          int64
	Pool           string
}

/*
 * https://redis.io/kb/doc/1p7wk5is89/how-can-i-sort-a-leaderboard-on-multiple-fields
 * encoding scores: 1. Ranking has an upper bound 99, 2. Score variable and unbounded
 */
func (s PoolScore) EncodeScore(withTvl bool) float64 {
	if withTvl {
		return float64(s.Level)*1e12 + s.LiquidityScore
	}

	return s.LiquidityScore
}

func GetMinScore(amountInUsd, threshold float64) (float64, error) {
	if amountInUsd < threshold {
		return float64(0), nil
	}

	return math.Floor(math.Log10(amountInUsd)) * 1e12, nil
}
