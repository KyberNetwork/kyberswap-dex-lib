package hashflowv3

import (
	"errors"
)

const (
	DexType = "hashflow-v3"

	Bps = 10000
)

var (
	defaultGas = Gas{Quote: 300000}

	ErrEmptyPriceLevels         = errors.New("empty price levels")
	ErrInsufficientLiquidity    = errors.New("insufficient liquidity")
	ErrAmtInLessThanMinAllowed  = errors.New("amountIn is less than min allowed")
	ErrAmtOutLessThanMinAllowed = errors.New("amountOut is less than min allowed")
)
