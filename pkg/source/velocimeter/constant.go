package velocimeter

const (
	DexTypeVelocimeter = "velocimeter"

	poolFactoryMethodAllPairLength = "allPairsLength"
	poolFactoryMethodAllPairs      = "allPairs"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"
	poolMethodGetFee      = "getFee"

	reserveZero         = "0"
	bps         float64 = 10000
)

var (
	DefaultGas = Gas{Swap: 227000}
)
