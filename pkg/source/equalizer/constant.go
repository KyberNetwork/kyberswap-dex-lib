package equalizer

const (
	DexTypeEqualizer = "equalizer"

	poolFactoryMethodAllPairLength = "allPairsLength"
	poolFactoryMethodAllPairs      = "allPairs"
	poolFactoryMethodGetRealFee    = "getRealFee"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"

	reserveZero = "0"
)

var DefaultGas = Gas{Swap: 227000}
