package shared

import "errors"

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidAmountIn  = errors.New("invalid amount in")
	ErrInvalidAmountOut = errors.New("invalid amount out")
)
