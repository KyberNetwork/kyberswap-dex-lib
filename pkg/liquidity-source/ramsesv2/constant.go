package ramsesv2

import (
	"math/big"
)

const (
	DexTypeRamsesV2      = "ramses-v2"
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	tickChunkSize        = 100
)

const (
	methodTicks = "ticks"
	methodV3Fee = "fee"

	methodV2GetLiquidity = "liquidity"
	methodV2GetSlot0     = "slot0"
	methodV2CurrentFee   = "currentFee"
	methodV2TickSpacing  = "tickSpacing"
)

var (
	zeroBI = big.NewInt(0)
)
