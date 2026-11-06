package dexv2

import (
	"github.com/pkg/errors"
)

var (
	ErrAdjustedAmountOutOfLimits  = errors.New("adjusted amount out of limits")
	ErrAmountOutOfLimits          = errors.New("amount out of limits")
	ErrFluidLiquidityCalcsError   = errors.New("fluid liquidity calcs error")
	ErrNextTickOutOfBounds        = errors.New("next tick out of bounds")
	ErrOverflow                   = errors.New("bigInt overflow int/uint256")
	ErrSqrtPriceChangeOutOfBounds = errors.New("sqrt price change out of bounds")
	ErrTokenReservesOverflow      = errors.New("token reserves overflow")
	ErrTokenReservesUnderflow     = errors.New("token reserves underflow")
	ErrUnsupportedLiquidityState  = errors.New("unsupported liquidity state or limit exceeded")
	ErrUnsupportedDynamicFee      = errors.New("unsupported dynamic fee")
	ErrUnsupportedController      = errors.New("unsupported controller")
	ErrV3TicksEmpty               = errors.New("v3 ticks is empty")
)
