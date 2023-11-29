package composablestable

import "errors"

var (
	ErrOverflow        = errors.New("overflow")
	ErrUnknownToken    = errors.New("unknown token")
	ErrInvalidReserve  = errors.New("invalid reserve")
	ErrReserveNotFound = errors.New("reserve not found")
	ErrPoolPaused      = errors.New("pool is paused")
)
