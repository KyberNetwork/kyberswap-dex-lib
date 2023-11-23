package stable

import "errors"

var (
	ErrTokenNotRegistered            = errors.New("TOKEN_NOT_REGISTERED")
	ErrInvalidReserve                = errors.New("invalid reserve")
	ErrInvalidAmountIn               = errors.New("invalid amount in")
	ErrStableGetBalanceDidntConverge = errors.New("stable get balance didn't converge")
	ErrInvalidPoolType               = errors.New("invalid pool type")
)
