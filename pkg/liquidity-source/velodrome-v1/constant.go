package velodromev1

const (
	DexType = "velodrome"
)

var (
	defaultGas = Gas{Swap: 227000}
)

const (
	pairFactoryMethodIsPaused       = "isPaused"
	pairFactoryMethodAllPairs       = "allPairs"
	pairFactoryMethodStableFee      = "stableFee"
	pairFactoryMethodVolatileFee    = "volatileFee"
	pairFactoryMethodAllPairsLength = "allPairsLength"
	pairFactoryMethodGetFee         = "getFee"

	stratumPairFactoryMethodGetFee = "getFee"

	nuriPairFactoryMethodPairFee = "pairFee"

	lyvePairFactoryMethodGetFee = "getFee"

	pairMethodMetadata    = "metadata"
	pairMethodGetReserves = "getReserves"
)

const (
	FeeTrackerIDVelodrome = "velodrome"
	FeeTrackerIDStratum   = "stratum"
	FeeTrackerIDNuri      = "nuri"
	FeeTrackerIDLyve      = "lyve"
)
