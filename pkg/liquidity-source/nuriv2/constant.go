package nuriv2

import (
	"math/big"
)

const (
	DexType              = "nuri-v2"
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	zeroString           = "0"
	emptyString          = ""
	tickChunkSize        = 100
)

const (
	methodGetLiquidity = "liquidity"
	methodGetSlot0     = "slot0"
	methodCurrentFee   = "currentFee"
	methodTickSpacing  = "tickSpacing"
	methodTicks        = "ticks"
)

var (
	zeroBI = big.NewInt(0)
)
