package algebrav1

import (
	"errors"
)

var (
	ErrUnmarshalVolLiq     = errors.New("failed to unmarshal volumePerLiquidityInBlock")
	ErrMaxBinarySearchLoop = errors.New("max binary search loop reached")
	ErrStaleTimepoints     = errors.New("getting stale timepoint data")
	ErrTickNil             = errors.New("tick is nil")
	ErrTicksEmpty          = errors.New("ticks list is empty")
	ErrInvalidToken        = errors.New("invalid token info")
	ErrZeroAmountIn        = errors.New("amountIn is 0")
	ErrZeroAmountOut       = errors.New("amountOut is 0")
	ErrSPL                 = errors.New("invalid sqrt price limit")
	ErrPoolLocked          = errors.New("pool is locked")
)
