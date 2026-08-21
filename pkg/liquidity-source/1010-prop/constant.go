package prop

import (
	"errors"
)

const (
	DexType    = "1010-prop"
	defaultGas = 135_000
)

var ErrInsufficientLiquidity = errors.New("insufficient liquidity")
