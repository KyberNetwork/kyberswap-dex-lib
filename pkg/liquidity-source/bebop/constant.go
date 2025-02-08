package bebop

import (
	"errors"
)

const DexType = "bebop"

var (
	defaultGas = Gas{Quote: 200000}
)

var (
	ErrEmptyPriceLevels      = errors.New("empty price levels")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
