package clipper

import (
	"errors"
)

const DexType = "clipper"

var defaultGas int64 = 80000

var (
	ErrInvalidTokenIn       = errors.New("invalid token in")
	ErrInvalidTokenOut      = errors.New("invalid token out")
	ErrInvalidPair          = errors.New("invalid pair")
	ErrFMVCheckFailed       = errors.New("FMV check failed")
	ErrAmountOutNaN         = errors.New("amountOut is NaN")
	ErrMinAmountInNotEnough = errors.New("minAmountIn is not enough")

	basisPoint float64 = 10000
)
