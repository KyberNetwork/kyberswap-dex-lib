package math

import "errors"

var (
	ErrAmount0DeltaOverflow = errors.New("amount0 delta overflow")
	ErrAmount1DeltaOverflow = errors.New("amount1 delta overflow")
	ErrNoLiquidity          = errors.New("no liquidity")
	ErrUnderflow            = errors.New("underflow")
	ErrOverflow             = errors.New("overflow")
	ErrMulDivOverflow       = errors.New("mul div overflow")
	ErrDivZero              = errors.New("division by 0")
	ErrWrongSwapDirection   = errors.New("wrong swap direction")
)
