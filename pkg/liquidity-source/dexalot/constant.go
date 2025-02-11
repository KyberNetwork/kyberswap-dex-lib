package dexalot

import (
	"errors"
)

const DexType = "dexalot"

var (
	defaultGas = Gas{Quote: 200000}
)

var (
	ErrEmptyPriceLevels                       = errors.New("empty price levels")
	ErrAmountInIsLessThanLowestPriceLevel     = errors.New("amountIn is less than lowest price level")
	ErrAmountInIsGreaterThanHighestPriceLevel = errors.New("amountIn is greater than highest price level")
	ErrNoSwapLimit                            = errors.New("swap limit is required for dexalot pools")
)
