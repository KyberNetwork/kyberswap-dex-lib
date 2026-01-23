package someswap

import "math/big"

const (
	DexType     = "someswap"
	reserveZero = "0"
)

const (
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodGetPair        = "allPairs"
	pairMethodToken0            = "token0"
	pairMethodToken1            = "token1"
	pairMethodGetReserves       = "getReserves"
	pairMethodFee               = "fee"
)

var (
	bpsDen    = big.NewInt(10000)
	weightDen = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	maxFeeBps = big.NewInt(9999)
)
