package mantisswap

import "errors"

var (
	ErrNoLp                = errors.New("no lp")
	ErrSwapNotAllowed      = errors.New("swap is not allowed")
	ErrLowAsset            = errors.New("low asset")
	ErrLargerThanMaxPower  = errors.New("larger than max power")
	ErrSmallerThanMinPower = errors.New("smaller than min power")
	ErrZeroAmount          = errors.New("toAmount is smaller than zero")
	ErrLpLimitReach        = errors.New("lp limit reached")
	ErrBeNegative          = errors.New("can not be negative")
	ErrPoolIsPaused        = errors.New("pool is paused")
)
