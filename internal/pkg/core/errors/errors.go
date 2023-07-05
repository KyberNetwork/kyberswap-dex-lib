package core

import "errors"

const ErrInvalidSwap = "invalid swap"

var (
	ErrZero                         = errors.New("zero")
	ErrBalancesMustMatchMultipliers = errors.New("balances must match multipliers")
	ErrDDoesNotConverge             = errors.New("d does not converge")
	ErrTokenFromEqualsTokenTo       = errors.New("can't compare token to itself")
	ErrTokenIndexesOutOfRange       = errors.New("token index out of range")
	ErrAmountOutNotConverge         = errors.New("approximation did not converge")
	ErrTokenNotFound                = errors.New("token not found")
	ErrWithdrawMoreThanAvailable    = errors.New("cannot withdraw more than available")

	ErrD1LowerThanD0                 = errors.New("d1 <= d0")
	ErrBasePoolExchangeNotSupported  = errors.New("not support exchange in base pool")
	ErrTokenToUnderLyingNotSupported = errors.New("not support exchange from base pool token to its underlying")
	ErrProvidersNotSupported         = errors.New("not support curve providers for this dex")

	ErrDenominatorZero = errors.New("denominator should not be 0")
)
