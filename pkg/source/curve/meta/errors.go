package meta

import "errors"

var (
	ErrInvalidBasePool               = errors.New("invalid base pool")
	ErrDDoesNotConverge              = errors.New("d does not converge")
	ErrTokenFromEqualsTokenTo        = errors.New("can't compare token to itself")
	ErrTokenIndexesOutOfRange        = errors.New("token index out of range")
	ErrAmountOutNotConverge          = errors.New("approximation did not converge")
	ErrBasePoolExchangeNotSupported  = errors.New("not support exchange in base pool")
	ErrTokenToUnderLyingNotSupported = errors.New("not support exchange from base pool token to its underlying")
)
