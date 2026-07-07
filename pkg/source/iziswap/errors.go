package iziswap

import "errors"

var (
	ErrLiquidityNil          = errors.New("liquidities is nil")
	ErrLimitOrderNil         = errors.New("limit Orders is nil")
	ErrInvalidReservesLength = errors.New("invalid reverses length")
	ErrInvalidTokensLength   = errors.New("invalid tokens length")
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmount         = errors.New("invalid amount")
)
