package getroute

import (
	"errors"
	"math"
)

const (
	ShrinkFuncNamePow       ShrinkFuncName = "pow"
	ShrinkFuncNameRound     ShrinkFuncName = "round"
	ShrinkFuncNameLogarithm ShrinkFuncName = "logarithm"
	ShrinkFuncNameDecimal   ShrinkFuncName = "decimal"
)

const (
	ShrinkFuncPowExp     ShrinkFuncConfig = "shrinkFuncPowExp"
	ShrinkDecimalBase    ShrinkFuncConfig = "shrinkDecimalBase"
	ShrinkFuncLogPercent ShrinkFuncConfig = "shrinkFuncLogPercent"
)

type ShrinkFuncName string

type ShrinkFuncConfig string

type ShrinkOption map[ShrinkFuncConfig]float64

type ShrinkFunc func(float64) float64

func ShrinkFuncFactory(name ShrinkFuncName, options ShrinkOption) (ShrinkFunc, error) {
	switch name {
	case ShrinkFuncNamePow:
		if powExp, ok := options[ShrinkFuncPowExp]; !ok {
			return nil, errors.New("option for ShrinkFuncPowExp has not been configured ShrinkFuncPowExp value")
		} else {
			return ShrinkFuncPow(powExp), nil
		}
	case ShrinkFuncNameRound:
		return ShrinkFuncRound, nil
	case ShrinkFuncNameDecimal:
		if decimalBase, ok := options[ShrinkDecimalBase]; !ok {
			return nil, errors.New("option for ShrinkDecimalBase has not been configured ShrinkDecimalBase value")
		} else {
			return func(v float64) float64 {
				l := math.Pow(decimalBase, math.Floor(math.Log(v)/math.Log(decimalBase)))
				return l * math.Round(v/l)
			}, nil
		}
	case ShrinkFuncNameLogarithm:
		if logPercent, ok := options[ShrinkFuncLogPercent]; !ok {
			return nil, errors.New("option for ShrinkFuncNameLogarithm has not been configured ShrinkFuncLogPercent value")
		} else {
			return func(f float64) float64 {
				return math.Pow(logPercent, math.Round(math.Log(f)/math.Log(logPercent)))
			}, nil
		}

	default:
		return ShrinkFuncRound, nil
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
