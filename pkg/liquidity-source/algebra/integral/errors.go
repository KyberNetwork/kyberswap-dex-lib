package integral

import (
	"errors"
)

var (
	ErrStaleTimepoints = errors.New("getting stale timepoint data")
	ErrTicksEmpty      = errors.New("ticks list is empty")
	ErrInvalidToken    = errors.New("invalid token info")
	ErrZeroAmountOut   = errors.New("amountOut is 0")

	ErrNotSupportFetchFullTick = errors.New("not support fetching full ticks")

	ErrIncorrectPluginFee    = errors.New("incorrect plugin fee")
	ErrInvalidLimitSqrtPrice = errors.New("invalid limit sqrt price")
	ErrNotInitialized        = errors.New("not initialized")
	ErrInvalidAmountRequired = errors.New("invalid amount required")
	ErrZeroAmountRequired    = errors.New("zero amount required")

	ErrLiquiditySub = errors.New("liquidity sub error")
	ErrLiquidityAdd = errors.New("liquidity add error")
	ErrOverflow     = errors.New("overflow")
	ErrUnderflow    = errors.New("underflow")
)
