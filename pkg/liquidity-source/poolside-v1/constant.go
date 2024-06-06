package poolsidev1

const (
	DexType = "poolside-v1"
)

var (
	defaultGas = Gas{Swap: 60000}
)

const (
	factoryMethodGetPair           = "allPairs"
	factoryMethodAllPairsLength    = "allPairsLength"
	pairMethodToken0               = "token0"
	pairMethodToken1               = "token1"
	pairMethodGetLiquidityBalances = "getLiquidityBalances"
)
