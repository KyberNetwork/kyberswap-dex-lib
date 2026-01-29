package someswapv2

import "math/big"

const (
	DexType     = "someswap-v2"
	reserveZero = "0"
)

const (
	factoryMethodAllPairsLength = "allPairsLength"
	factoryEventPairCreated     = "PairCreated"

	poolMethodToken0      = "token0"
	poolMethodToken1      = "token1"
	poolMethodReserve0    = "reserve0"
	poolMethodReserve1    = "reserve1"
	poolMethodGetReserves = "getReserves"
)

var (
	feeDen    = big.NewInt(1000000)
	weightDen = new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
)

