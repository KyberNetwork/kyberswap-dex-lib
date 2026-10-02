package solidlyv2

const (
	DexType = "solidly-v2"
)

var (
	defaultGas = Gas{Swap: 227000}
)

const (
	factoryMethodIsPaused       = "isPaused"
	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodStableFees     = "stableFees"
	factoryMethodVolatileFees   = "volatileFees"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"
	poolMethodFeeRatio    = "feeRatio"

	memecoreMethodPoolFee = "poolFee"
	memecoreMethodToken0  = "token0"
	memecoreMethodToken1  = "token1"

	shadowLegacyMethodFee = "fee"
)
