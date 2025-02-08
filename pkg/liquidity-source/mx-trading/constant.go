package mxtrading

import (
	"errors"
)

const DexType = "mx-trading"

var (
	defaultGas = Gas{FillOrderArgs: 180000}
)

var (
	ErrEmptyPriceLevels                    = errors.New("empty price levels")
	ErrAmountInIsLessThanLowestPriceLevel  = errors.New("amountIn is less than lowest price level")
	ErrAmountInIsGreaterThanTotalLevelSize = errors.New("amountIn is greater than total level size")
	ErrAmountOutIsGreaterThanInventory     = errors.New("amountOut is greater than inventory")
)
