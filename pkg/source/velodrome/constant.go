package velodrome

const (
	DexTypeVelodrome   = "velodrome"
	defaultTokenWeight = 50

	poolFactoryMethodAllPairLength = "allPairsLength"
	poolFactoryMethodAllPairs      = "allPairs"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"
	poolMethodStableFee   = "stableFee"
	poolMethodVolatileFee = "volatileFee"

	reserveZero         = "0"
	bps         float64 = 10000
)

var (
	DefaultGas = Gas{Swap: 227000}
)
