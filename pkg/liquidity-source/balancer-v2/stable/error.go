package stable

import "errors"

var (
	ErrTokenNotRegistered = errors.New("TOKEN_NOT_REGISTERED")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrInvalidPoolType    = errors.New("invalid pool type")
	ErrInvalidPoolID      = errors.New("invalid pool id")
)
