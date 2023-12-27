package stable

const (
	DexType = "balancer-v2-stable"

	poolTypeStable     = "Stable"
	poolTypeMetaStable = "MetaStable"

	poolTypeVer1 = 1
	poolTypeVer2 = 2

	poolMethodGetSwapFeePercentage      = "getSwapFeePercentage"
	poolMethodGetPausedState            = "getPausedState"
	poolMethodGetAmplificationParameter = "getAmplificationParameter"
	poolMethodGetVault                  = "getVault"
	poolMethodGetScalingFactors         = "getScalingFactors"

	defaultWeight = 1
)

var (
	defaultGas = Gas{Swap: 80000}
)
