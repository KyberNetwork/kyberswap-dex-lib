package weighted

import "time"

const (
	DexType = "balancer-weighted"

	poolTypeWeighted = "Weighted"
)

const (
	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
)

const (
	graphQLRequestTimeout = 20 * time.Second

	poolTypeVer1 = 1
)
