package hashflowv3

import (
	"errors"
	"math/big"
)

const DexType = "hashflow-v3"

var (
	zeroBF            = big.NewFloat(0)
	defaultGas        = Gas{Quote: 300000}
	priceToleranceBps = 10000
)

var (
	ErrEmptyPriceLevels                        = errors.New("empty price levels")
	ErrInsufficientLiquidity                   = errors.New("insufficient liquidity")
	ErrParsingBigFloat                         = errors.New("invalid float number")
	ErrAmountInIsLessThanLowestPriceLevel      = errors.New("amountIn is less than lowest price level")
	ErrAmountInIsGreaterThanHighestPriceLevel  = errors.New("amountIn is greater than highest price level")
	ErrAmountOutIsLessThanLowestPriceLevel     = errors.New("amountOut is less than lowest price level")
	ErrAmountOutIsGreaterThanHighestPriceLevel = errors.New("amountOut is greater than highest price level")
)
