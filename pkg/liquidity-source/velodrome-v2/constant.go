package velodromev2

const (
	DexType = "velodrome-v2"
)

var (
	defaultGas = Gas{Swap: 227000}
)

const (
	poolFactoryMethodIsPaused       = "isPaused"
	poolFactoryMethodAllPools       = "allPools"
	poolFactoryMethodAllPoolsLength = "allPoolsLength"
	poolFactoryMethodGetFee         = "getFee"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"
)
