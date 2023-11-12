package velodrome

const (
	DexType = "velodrome"
)

var (
	defaultGas = Gas{Swap: 227000}
)

const (
	pairFactoryMethodIsPaused       = "isPaused"
	pairFactoryMethodGetPair        = "allPairs"
	pairFactoryMethodStableFee      = "stableFee"
	pairFactoryMethodVolatileFee    = "volatileFee"
	pairFactoryMethodAllPairsLength = "allPairsLength"
	pairFactoryMethodGetFee         = "getFee"

	pairMethodMetadata    = "metadata"
	pairMethodGetReserves = "getReserves"
)
