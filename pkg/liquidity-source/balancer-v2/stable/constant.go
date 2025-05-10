package stable

const (
	DexType = "balancer-v2-stable"

	poolTypeLegacyStable     = "Stable"
	poolTypeLegacyMetaStable = "MetaStable"

	poolTypeStable     = "STABLE"
	poolTypeMetaStable = "META_STABLE"

	poolTypeVer1 = 1
	poolTypeVer2 = 2

	poolMethodGetSwapFeePercentage      = "getSwapFeePercentage"
	poolMethodGetPausedState            = "getPausedState"
	poolMethodGetAmplificationParameter = "getAmplificationParameter"
	poolMethodGetVault                  = "getVault"
	poolMethodGetScalingFactors         = "getScalingFactors"
	poolMethodGetPoolId                 = "getPoolId"

	protocolMethodGetSwapFeePercentage = "getSwapFeePercentage"

	defaultWeight = 1

	poolSpecializationGeneral = 0
)

var (
	defaultGas = Gas{Swap: 80000}
)
