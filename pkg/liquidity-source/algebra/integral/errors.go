package integral

import (
	"errors"
)

var (
	ErrStaleTimepoints      = errors.New("getting stale timepoint data")
	ErrTicksEmpty           = errors.New("ticks list is empty")
	ErrInvalidToken         = errors.New("invalid token info")
	ErrZeroAmountCalculated = errors.New("zero amount calculated")

	ErrNotSupportFetchFullTick = errors.New("not support fetching full ticks")

	ErrIncorrectPluginFee     = errors.New("incorrect plugin fee")
	ErrInvalidLimitSqrtPrice  = errors.New("invalid limit sqrt price")
	ErrTargetIsTooOld         = errors.New("target is too old")
	ErrNotInitialized         = errors.New("not initialized")
	ErrPoolLocked             = errors.New("pool has been locked and not usable")
	ErrInvalidAmountRequired  = errors.New("invalid amount required")
	ErrZeroAmountRequired     = errors.New("zero amount required")
	ErrZeroPrice              = errors.New("price cannot be zero")
	ErrZeroLiquidity          = errors.New("liquidity cannot be zero")
	ErrInvalidPriceUpperLower = errors.New("price upper must not be less than price lower")
	ErrInvalidPriceLower      = errors.New("price lower must be positive")

	ErrLiquiditySub = errors.New("liquidity sub error")
	ErrLiquidityAdd = errors.New("liquidity add error")
	ErrOverflow     = errors.New("overflow")
	ErrUnderflow    = errors.New("underflow")
)
