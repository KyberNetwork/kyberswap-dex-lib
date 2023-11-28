package stable

const (
	DexType = "balancer-v2-stable"

	poolTypeStable     = "Stable"
	poolTypeMetaStable = "MetaStable"

	poolTypeVersion1 = 1
	poolTypeVersion2 = 2

	poolMethodGetSwapFeePercentage      = "getSwapFeePercentage"
	poolMethodGetPausedState            = "getPausedState"
	poolMethodGetAmplificationParameter = "getAmplificationParameter"

	defaultWeight = 1
)

var (
	defaultGas = Gas{Swap: 10}
)
