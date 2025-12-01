package dexv2

import (
	"github.com/pkg/errors"
)

var (
	ErrUnsupportedController     = errors.New("unsupported controller")
	ErrAmountOutOfLimits         = errors.New("amount out of limits")
	ErrAdjustedAmountOutOfLimits = errors.New("adjusted amount out of limits")
	ErrFluidLiquidityCalcsError  = errors.New("fluid liquidity calcs error")
	ErrNextTickOutOfBounds       = errors.New("next tick out of bounds")
)
