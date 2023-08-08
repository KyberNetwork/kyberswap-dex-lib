package velocimeter

import "time"

const (
	DexTypeVelocimeter         = "velocimeter"
	defaultNewPoolLimit       = 100
	defaultNewPoolJobInterval = 600 * time.Second
	defaultTokenWeight        = 50

	poolFactoryMethodAllPairLength = "allPairsLength"
	poolFactoryMethodAllPairs      = "allPairs"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"
	poolMethodGetFee   = "getFee"

	reserveZero         = "0"
	bps         float64 = 10000
)

var (
	DefaultGas = Gas{Swap: 227000}
)
