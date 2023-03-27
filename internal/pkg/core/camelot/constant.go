package camelot

import (
	"errors"
)

var (
	DefaultGas = Gas{Swap: 128000}
)

var (
	ErrInsufficientOutputAmount = errors.New("CamelotPair: INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("CamelotPair: INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                 = errors.New("CamelotPair: K")
)
