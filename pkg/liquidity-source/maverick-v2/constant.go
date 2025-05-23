package maverickv2

const (
	DexType = "maverick-v2"
)

const (
	factoryMethodLookup = "lookup"

	poolMethodTokenA   = "tokenA"
	poolMethodTokenB   = "tokenB"
	poolMethodGetState = "getState"
)

const (
	GasSwap         = int64(125000)
	GasCrossBin     = int64(20000)
	MaxSwapCalcIter = 150
	// Constants matching TypeScript implementation
	MAX_TICK = 460540
)
