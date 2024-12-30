package stable

const (
	DexType = "balancer-v3-stable"

	PoolType = "StablePool"

	poolMethodGetStablePoolDynamicData   = "getStablePoolDynamicData"
	poolMethodGetStablePoolImmutableData = "getStablePoolImmutableData"
)

var (
	defaultGas = Gas{Swap: 80000}
)
