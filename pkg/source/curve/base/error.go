package base

import "errors"

var (
	ErrInvalidAValue                = errors.New("invalid A value")
	ErrZero                         = errors.New("zero")
	ErrBalancesMustMatchMultipliers = errors.New("balances must match multipliers")
	ErrDDoesNotConverge             = errors.New("d does not converge")
	ErrTokenFromEqualsTokenTo       = errors.New("can't compare token to itself")
	ErrTokenIndexesOutOfRange       = errors.New("token index out of range")
	ErrAmountOutNotConverge         = errors.New("approximation did not converge")
	ErrTokenNotFound                = errors.New("token not found")
	ErrWithdrawMoreThanAvailable    = errors.New("cannot withdraw more than available")
	ErrD1LowerThanD0                = errors.New("d1 <= d0")
)
