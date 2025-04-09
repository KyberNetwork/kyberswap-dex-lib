package shared

import "errors"

var (
	ErrInvalidExtra     = errors.New("invalid extra data")
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidAmountIn  = errors.New("invalid amount in")
	ErrInvalidAmountOut = errors.New("invalid amount out")
)
