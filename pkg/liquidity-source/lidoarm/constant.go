package lidoarm

import "errors"

const (
	DexType         = "lidoarm"
	defaultReserves = "100000000000000000000000"

	defaultGas = 50000
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
)
