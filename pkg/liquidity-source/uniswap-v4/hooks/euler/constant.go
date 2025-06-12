package euler

import (
	"errors"
)

const (
	DexType = "uniswap-v4-euler"

	DefaultGas int64 = 400000

	factoryMethodPools = "pools"

	poolMethodGetAssets   = "getAssets"
	poolMethodGetReserves = "getReserves"
	poolMethodGetParams   = "getParams"
)

var (
	ErrInvalidVaults        = errors.New("invalid vaults")
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidReserve       = errors.New("invalid reserve")
	ErrInvalidAmountIn      = errors.New("invalid amount in")
	ErrInvalidAmountOut     = errors.New("invalid amount out")
	ErrSwapIsPaused         = errors.New("swap is paused")
	ErrOverflow             = errors.New("math overflow")
	ErrCurveViolation       = errors.New("curve violation")
	ErrOperatorNotInstalled = errors.New("operator not installed")
	ErrDivisionByZero       = errors.New("division by zero")
	ErrSwapLimitExceeded    = errors.New("swap limit exceed")
)
