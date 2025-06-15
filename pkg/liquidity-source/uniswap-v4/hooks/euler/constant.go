package euler

import (
	"errors"
)

const (
	DexType = "uniswap-v4-euler"

	DefaultGas int64 = 400000

	tickSpacing int32 = 1 // hard-coded tick spacing, as its unused

	factoryMethodPools = "pools"

	poolMethodGetAssets   = "getAssets"
	poolMethodGetReserves = "getReserves"
	poolMethodGetParams   = "getParams"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidAmountIn   = errors.New("invalid amount in")
	ErrInvalidAmountOut  = errors.New("invalid amount out")
	ErrSwapIsPaused      = errors.New("swap is paused")
	ErrOverflow          = errors.New("math overflow")
	ErrCurveViolation    = errors.New("curve violation")
	ErrDivisionByZero    = errors.New("division by zero")
	ErrSwapLimitExceeded = errors.New("swap limit exceed")
)
