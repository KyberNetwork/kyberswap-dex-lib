package solidlyv3

import (
	"math/big"
)

const (
	DexTypeSolidlyV3     = "solidly-v3"
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	zeroString           = "0"
	emptyString          = ""
	tickChunkSize        = 100
)

const (
	methodGetLiquidity = "liquidity"
	methodGetSlot0     = "slot0"
	methodTickSpacing  = "tickSpacing"
	methodTicks        = "ticks"
)

var (
	zeroBI = big.NewInt(0)
)
