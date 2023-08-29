package velodromev2

const (
	DexTypeVelodromeV2 = "velodrome-v2"
	defaultTokenWeight = 50

	poolFactoryMethodAllPoolsLength = "allPoolsLength"
	poolFactoryMethodAllPools       = "allPools"
	factoryMethodGetFee             = "getFee"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"

	reserveZero         = "0"
	bps         float64 = 10000
)

var (
	DefaultGas = Gas{Swap: 227000}
)
