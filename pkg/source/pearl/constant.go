package pearl

const (
	DexTypePearl = "pearl"

	poolFactoryMethodAllPairLength = "allPairsLength"
	poolFactoryMethodAllPairs      = "allPairs"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"
	poolMethodStableFee   = "stableFee"
	poolMethodVolatileFee = "volatileFee"

	reserveZero = "0"
)

var (
	DefaultGas = Gas{Swap: 227000}

	feePrecision int64 = 1e18
)
