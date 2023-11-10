package uniswapv2

import "github.com/holiman/uint256"

const (
	DexType = "uniswap-v2"
)

var (
	defaultGas = Gas{Swap: 60000}
)

const (
	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
)

const (
	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"
)

var zero = uint256.NewInt(0)
