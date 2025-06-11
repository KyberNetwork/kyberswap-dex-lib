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

	MaxTick = 460540 // MaxTick is the maximum tick value for Maverick V2 pools.

	GasSwap     = 125000
	GasCrossBin = 20000
)

var (
	DefaultBinBatchSize = 500

	ErrEmptyBins = errors.New("empty bins")
	ErrOverflow  = errors.New("overflow")
)
