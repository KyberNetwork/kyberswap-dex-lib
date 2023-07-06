package platypus

import "errors"

var (
	ErrInvalidOracleType = errors.New("invalid oracle type")
	ErrDivisionByZero    = errors.New("division by zero")
	ErrPoolPaused        = errors.New("pool is paused")
	ErrWETHNotFound      = errors.New("weth not found")

	// ErrSameAddress swapping with tokenIn = tokenOut
	ErrSameAddress      = errors.New("SAME_ADDRESS")
	ErrAssetNotExist    = errors.New("ASSET_NOT_EXIST")
	ErrDiffAggAcc       = errors.New("DIFF_AGG_ACC")
	ErrZeroFromAmount   = errors.New("ZERO_FROM_AMOUNT")
	ErrInsufficientCash = errors.New("INSUFFICIENT_CASH")
	ErrUnsupportedSwap  = errors.New("UNSUPPORTED_SWAP")
)
