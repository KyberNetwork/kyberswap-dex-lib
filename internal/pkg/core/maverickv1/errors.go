package maverickv1

import "errors"

var (
	ErrLargerThanMaxTick = errors.New("tick is larger than max tick")
	ErrMulOverflow       = errors.New("mul overflow")
	ErrDividedByZero     = errors.New("divided by zero")
	ErrInvalidLiquidity  = errors.New("invalid liquidity")
	ErrMaxSwapIteration  = errors.New("reached max swap iteration calculation")
)
