package usdfi

const (
	DexTypeUSDFi       = "usdfi"
	defaultTokenWeight = 50

	poolFactoryMethodAllPairLength   = "allPairsLength"
	poolFactoryMethodAllPairs        = "allPairs"
	poolFactoryMethodBaseStableFee   = "baseStableFee"
	poolFactoryMethodBaseVariableFee = "baseVariableFee"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"

	reserveZero = "0"
)

var (
	DefaultGas = Gas{Swap: 227000}

	numeratorOne = 1
)
