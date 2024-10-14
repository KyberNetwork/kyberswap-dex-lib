package ringswapv2

const (
	DexType     = "ringswap-v2"
	ZeroAddress = "0x0000000000000000000000000000000000000000"
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
	zkSwapFinancePairMethodGetSwapFee          = "getSwapFee"

	fewWrappedTokenGetTokenMethod = "token"
)

const (
	FeeTrackerIDMMF         = "mmf"
	FeeTrackerIDMdex        = "mdex"
	FeeTrackerIDShibaswap   = "shibaswap"
	FeeTrackerIDDefiswap    = "defiswap"
	FeeTrackerZKSwapFinance = "zkswap-finance"
)
