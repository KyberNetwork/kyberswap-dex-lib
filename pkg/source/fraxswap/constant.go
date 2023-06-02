package fraxswap

const (
	DexTypeFraxswap    = "fraxswap"
	defaultTokenWeight = 50

	poolFactoryMethodAllPairsLength = "allPairsLength"
	poolFactoryMethodAllPairs       = "allPairs"

	poolMethodToken0               = "token0"
	poolMethodToken1               = "token1"
	poolMethodGetReserveAfterTwamm = "getReserveAfterTwamm"
	poolMethodFee                  = "fee"

	reserveZero = "0"
)

var (
	DefaultGas = Gas{Swap: 113276}
)
