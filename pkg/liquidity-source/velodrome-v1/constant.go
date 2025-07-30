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
	pairFactoryMethodAllPairsLength = "allPairsLength"
	pairMethodMetadata              = "metadata"
	pairMethodGetReserves           = "getReserves"

	genericMethodFee        = "fee"
	genericTemplatePool     = "pool"
	genericTemplateFactory  = "factory"
	genericTemplateIsStable = "isStable"
)
