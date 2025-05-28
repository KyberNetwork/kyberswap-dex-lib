package maverickv2

import "math/big"

const (
	DexType = "maverick-v2"
)

const (
	factoryMethodLookup = "lookup"

	poolMethodTokenA      = "tokenA"
	poolMethodTokenB      = "tokenB"
	poolMethodGetState    = "getState"
	poolMethodTickSpacing = "tickSpacing"

	poolLensMethodGetFullPoolState = "getFullPoolState"
)

const (
	GasSwap             = int64(125000)
	GasCrossBin         = int64(20000)
	DefaultBinBatchSize = 5000
)

// MAX_TICK is the maximum tick value for Maverick V2 pools.
var MAX_TICK = big.NewInt(460540)
