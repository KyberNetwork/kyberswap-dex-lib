package shared

import "errors"

var (
	ErrUnsupportedHook   = errors.New("unsupported hook")
	ErrUninitializedPool = errors.New("pool is uninitialized")
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidReserve    = errors.New("invalid reserve")
	ErrInvalidAmountIn   = errors.New("invalid amount in")
	ErrInvalidParameters = errors.New("invalid parameters")
)
