package weighted

import "time"

const (
	DexType = "balancer-weighted"

	poolTypeWeighted = "Weighted"
)

const (
	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetScalingFactors    = "getScalingFactors"
)

const (
	graphQLRequestTimeout = 20 * time.Second
)
