package algebrav1

import (
	"errors"
)

var (
	ErrUnmarshalVolLiq = errors.New("failed to unmarshal volumePerLiquidityInBlock")
	ErrMaxBinarySearchLoop = errors.New("max binary search loop reached")
	ErrStaleTimepoints = errors.New("getting stale timepoint data")
	ErrFeeNotFound = errors.New("cannot find fee info")
)
