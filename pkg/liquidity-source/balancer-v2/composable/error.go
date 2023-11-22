package composable

import "errors"

var (
	ErrOverflow       = errors.New("overflow")
	ErrUnknownToken   = errors.New("unknown token")
	ErrInvalidReserve = errors.New("invalid reserve")
)
