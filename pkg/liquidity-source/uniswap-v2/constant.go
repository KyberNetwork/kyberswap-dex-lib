package uniswapv2

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

	meerkatPairMethodSwapFee                   = "swapFee"
	mdexFactoryMethodGetPairFees               = "getPairFees"
	shibaswapPairMethodTotalFee                = "totalFee"
	croDefiSwapFactoryMethodTotalFeeBasisPoint = "totalFeeBasisPoint"
)

const (
	FeeTrackerIDMMF       = "mmf"
	FeeTrackerIDMdex      = "mdex"
	FeeTrackerIDShibaswap = "shibaswap"
	FeeTrackerIDDefiswap  = "defiswap"
)
