package maverickv1

import "errors"

var (
	ErrLargerThanMaxTick = errors.New("tick is larger than max tick")
	ErrOverflow          = errors.New("overflow")
	ErrMulOverflow       = errors.New("mul overflow")
	ErrDividedByZero     = errors.New("divided by zero")
	ErrInvalidLiquidity  = errors.New("invalid liquidity")
	ErrInvalidDeltaOut   = errors.New("invalid delta out") // L
	ErrEmptyBins         = errors.New("maverick pool has no bin")
	ErrEmptyBinMap       = errors.New("maverick bin map is empty")
)
