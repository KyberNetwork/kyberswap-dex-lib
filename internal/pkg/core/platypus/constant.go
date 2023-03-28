package platypus

import (
	"errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

var (
	DefaultGas = Gas{Swap: 88000}
)

var (
	ErrPoolPaused     = errors.New("pool is paused")
	ErrWETHNotFound   = errors.New("weth not found")
	ErrDivisionByZero = errors.New("division by zero")

	// ErrSameAddress swapping with tokenIn = tokenOut
	ErrSameAddress      = errors.New("SAME_ADDRESS")
	ErrAssetNotExist    = errors.New("ASSET_NOT_EXIST")
	ErrDiffAggAcc       = errors.New("DIFF_AGG_ACC")
	ErrZeroFromAmount   = errors.New("ZERO_FROM_AMOUNT")
	ErrInsufficientCash = errors.New("INSUFFICIENT_CASH")
	ErrUnsupportedSwap  = errors.New("UNSUPPORTED_SWAP")
)

var (
	WAD = constant.TenPowInt(18)
	RAY = constant.TenPowInt(27)
)
