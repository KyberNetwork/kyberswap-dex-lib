package iziswap

import "errors"

var (
	ErrLiquidityNil          = errors.New("liquidities is nil")
	ErrLimitOrderNil         = errors.New("limit Orders is nil")
	ErrInvalidReservesLength = errors.New("invalid reverses length")
	ErrInvalidTokensLength   = errors.New("invalid tokens length")
	ErrInvalidReserve        = errors.New("invalid reserve")
)
