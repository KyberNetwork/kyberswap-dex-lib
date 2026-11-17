package maverickv2

import (
	"errors"
)

const (
	DexType = "maverick-v2"

	factoryMethodLookup = "lookup"

	poolMethodTokenA   = "tokenA"
	poolMethodTokenB   = "tokenB"
	poolMethodGetState = "getState"

	poolLensMethodGetFullPoolState = "getFullPoolState"

	threshold = 5e7

	GasSwap     = 125000
	GasCrossBin = 20000
	// GasEmptyTick is the gas per tick without liquidity that a swap walks over: 2,541-2,650 gas per tick on
	// the ezETH/WETH pool 0x707e…f171 at block 26299905, measured on a fork with bounded tick limits.
	GasEmptyTick = 2600
)

var (
	DefaultBinBatchSize = 500

	ErrEmptyBins = errors.New("empty bins")
	ErrOverflow  = errors.New("overflow")

	ErrInsufficientLiquidity = errors.New("INSUFFICIENT_LIQUIDITY")
)
