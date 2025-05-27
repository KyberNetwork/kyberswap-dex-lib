package maverickv2

import "math/big"

const (
	DexType = "maverick-v2"

	// API URL for Maverick V2 API
	MaverickAPIURL = "https://v2-api.mav.xyz"
)

const (
	factoryMethodLookup = "lookup"

	poolMethodTokenA      = "tokenA"
	poolMethodTokenB      = "tokenB"
	poolMethodGetState    = "getState"
	poolMethodTickSpacing = "tickSpacing"
)

const (
	GasSwap     = int64(125000)
	GasCrossBin = int64(20000)
)

// MAX_TICK is the maximum tick value for Maverick V2 pools.
var MAX_TICK = big.NewInt(460540)
