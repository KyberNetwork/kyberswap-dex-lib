package ethervista

const (
	DexType = "ether-vista"
)

var (
	defaultGas = Gas{Swap: 60000}
)

const (
	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodRouter         = "router"
)

const (
	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"
)
