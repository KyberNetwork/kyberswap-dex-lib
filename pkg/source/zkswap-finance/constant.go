package zkswapfinance

const (
	DexTypeZkSwapFinance = "zkswap-finance"

	defaultTokenWeight = 50
	defaultSwapFee     = "0.003" // (30 / 10000)
	reserveZero        = "0"
)

const (
	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"

	pairMethodToken0            = "token0"
	pairMethodToken1            = "token1"
	pairMethodGetReservesSimple = "getReservesSimple"
	pairMethodGetSwapFee        = "getSwapFee"
)

const (
	bps = 10000
)

var (
	defaultGas = Gas{SwapBase: 60000, SwapNonBase: 102000}
)
