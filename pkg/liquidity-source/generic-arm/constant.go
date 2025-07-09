package genericarm

import "errors"

const (
	DexType         = "generic-arm"
	defaultReserves = "100000000000000000000000"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrUnsupportedSwap         = errors.New("unsupported swap")
	ErrUnsupportedArmType      = errors.New("unsupported arm type")
)
