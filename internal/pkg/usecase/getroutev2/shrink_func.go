package getroutev2

import (
	"math"
)

const (
	ShrinkFuncNamePow   = "pow"
	ShrinkFuncNameRound = "round"
)

type ShrinkFunc func(float64) float64

func ShrinkFuncFactory(config CacheConfig) ShrinkFunc {
	switch config.ShrinkFuncName {
	case ShrinkFuncNamePow:
		return ShrinkFuncPow(config.ShrinkFuncPowExp)
	case ShrinkFuncNameRound:
		return ShrinkFuncRound
	default:
		return ShrinkFuncRound
	}
}

func ShrinkFuncPow(exp float64) ShrinkFunc {
	return func(v float64) float64 {
		return math.Pow(v, exp)
	}
}

func ShrinkFuncRound(v float64) float64 {
	return math.Round(v)
}
