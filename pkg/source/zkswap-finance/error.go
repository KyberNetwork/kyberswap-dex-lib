package zkswapfinance

import "errors"

var (
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
)
