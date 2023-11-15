package velodromev1

const (
	DexType = "velodrome"
)

var (
	defaultGas = Gas{Swap: 227000}
)

const (
	pairFactoryMethodIsPaused       = "isPaused"
	pairFactoryMethodAllPairs       = "allPairs"
	pairFactoryMethodStableFee      = "stableFee"
	pairFactoryMethodVolatileFee    = "volatileFee"
	pairFactoryMethodAllPairsLength = "allPairsLength"
	pairFactoryMethodGetFee         = "getFee"

	pairMethodMetadata    = "metadata"
	pairMethodGetReserves = "getReserves"
)
