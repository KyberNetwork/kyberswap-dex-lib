package getroute

import (
	"math"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	ShrinkFuncNamePow       = "pow"
	ShrinkFuncNameRound     = "round"
	ShrinkFuncNameLogarithm = "logarithm"
	ShrinkFuncNameDecimal   = "decimal"
)

type ShrinkFunc func(float64) float64

func ShrinkFuncFactory(config valueobject.CacheConfig) ShrinkFunc {
	switch config.ShrinkFuncName {
	case ShrinkFuncNamePow:
		return ShrinkFuncPow(config.ShrinkFuncPowExp)
	case ShrinkFuncNameRound:
		return ShrinkFuncRound
	case ShrinkFuncNameDecimal:
		return ShrinkFuncDecimal
	case ShrinkFuncNameLogarithm:
		return func(f float64) float64 {
			return math.Pow(config.ShrinkFuncLogPercent, math.Round(math.Log(f)/math.Log(config.ShrinkFuncLogPercent)))
		}
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

func ShrinkFuncDecimal(v float64) float64 {
	l := math.Pow10(int(math.Floor(math.Log10(v))))
	return l * math.Round(v/l)
}
