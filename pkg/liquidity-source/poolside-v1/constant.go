package poolsidev1

const (
	DexType = "poolside-v1"
)

var (
	defaultGas      = Gas{Swap: 60000}
	defaultDecimals = uint8(18)
)

const (
	factoryMethodGetPair                = "allPairs"
	factoryMethodAllPairsLength         = "allPairsLength"
	pairMethodToken0                    = "token0"
	pairMethodToken1                    = "token1"
	pairMethodGetLiquidityBalances      = "getLiquidityBalances"
	buttonTokenMethodGetUnderlyingToken = "underlying"
	erc20TokenDecimals                  = "decimals"
)
