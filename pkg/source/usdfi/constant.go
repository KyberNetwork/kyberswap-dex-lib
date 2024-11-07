package usdfi

const (
	DexTypeUSDFi       = "usdfi"
	defaultTokenWeight = 50

	poolFactoryMethodAllPairLength = "allPairsLength"
	poolFactoryMethodAllPairs      = "allPairs"
	poolFactoryMethodIsPaused      = "isPaused"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"
	poolMethodFee         = "fee"

	reserveZero = "0"
)

var (
	DefaultGas = Gas{Swap: 227000}

	numeratorOne = 1
)
