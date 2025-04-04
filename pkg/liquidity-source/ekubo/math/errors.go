package math

import "errors"

var (
	ErrNoLiquidity        = errors.New("no liquidity")
	ErrUnderflow          = errors.New("underflow")
	ErrOverflow           = errors.New("overflow")
	ErrDivZero            = errors.New("division by 0")
	ErrWrongSwapDirection = errors.New("wrong swap direction")
)
