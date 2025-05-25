package weighted

const (
	DexType = "balancer-v2-weighted"

	poolTypeLegacyWeighted = "Weighted"
	poolTypeWeighted       = "WEIGHTED"

	poolTypeVer1 = 1

	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
	poolMethodGetVault             = "getVault"

	poolMethodTotalSupply      = "totalSupply"
	poolMethodGetInvariant     = "getInvariant"
	poolMethodGetLastInvariant = "getLastInvariant"

	protocolMethodGetSwapFeePercentage = "getSwapFeePercentage"
)

var (
	defaultGas = Gas{Swap: 80535}
)
