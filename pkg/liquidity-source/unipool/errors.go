package unipool

import "errors"

var (
	ErrInvalidToken              = errors.New("invalid token")
	ErrInvalidReserve            = errors.New("invalid reserve")
	ErrInvalidAmountIn           = errors.New("invalid amount in")
	ErrInvalidAmountOut          = errors.New("invalid amount out")
	ErrInsufficientLiquidity     = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrInsufficientOutputAmount  = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientSwapLiquidity = errors.New("INSUFFICIENT_SWAP_LIQUIDITY")
	ErrFeeExceedsMax             = errors.New("FEE_EXCEEDS_MAX")
	ErrExcessiveSpread           = errors.New("EXCESSIVE_SPREAD")
)
