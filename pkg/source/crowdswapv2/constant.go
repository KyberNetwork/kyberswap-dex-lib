package crowdswapv2

import "math/big"

const (
	DexTypeCrowdswapV2 = "crowdswapv2"
	defaultTokenWeight = 50
	reserveZero        = "0"
)

const (
	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	pairMethodToken0            = "token0"
	pairMethodToken1            = "token1"
	pairMethodGetSwapFee        = "swapFee"
	pairMethodGetReserves       = "getReserves"
)

var (
	zeroBI         = big.NewInt(0)
	defaultGas     = Gas{SwapBase: 60000, SwapNonBase: 102000}
	defaultSwapFee = "2"
	bOne           = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	bOneFloat, _   = new(big.Float).SetString("1000000000000000000")
)
