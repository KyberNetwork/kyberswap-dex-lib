package stable

const (
	DexType = "balancer-v3-stable"

	SubgraphPoolType = "STABLE"

	poolMethodGetAmplificationParameter = "getAmplificationParameter"

	stableSurgeHookMethodGetMaxSurgeFeePercentage    = "getMaxSurgeFeePercentage"
	stableSurgeHookMethodGetSurgeThresholdPercentage = "getSurgeThresholdPercentage"
)

var (
	baseGas   int64 = 237494
	bufferGas int64 = 120534
)
