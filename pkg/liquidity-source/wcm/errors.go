package wcm

import "errors"

var (
	ErrEmptyOrderBook        = errors.New("empty order book")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrInvalidTokenPair      = errors.New("invalid token pair")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInvalidAmountOut      = errors.New("invalid amount out")
	ErrAmountOutTooSmall     = errors.New("amount out too small")
	ErrPoolHalted            = errors.New("pool halted")
	ErrQuantityTooLow        = errors.New("quantity too low")
)
