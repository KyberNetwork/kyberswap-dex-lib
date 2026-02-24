package shared

import "errors"

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrInvalidAmountIn     = errors.New("invalid amount in")
	ErrInvalidAmountOut    = errors.New("invalid amount out")
	ErrInsufficientReserve = errors.New("insufficient reserve")
	ErrSwapIsPaused        = errors.New("swap is paused")
	ErrSwapExpired         = errors.New("swap expired")
	ErrMultiDebts          = errors.New("multiple debts")
	ErrInsolvency          = errors.New("insolvency")
	ErrCurveViolation      = errors.New("curve violation")
	ErrSwapLimitExceeded   = errors.New("swap limit exceed")
	ErrSwapRejected        = errors.New("swap rejected")
	ErrDivisionByZero      = errors.New("division by zero")
)
