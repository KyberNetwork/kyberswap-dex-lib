package weighted

import "time"

const (
	DexTypeBalancerWeighted = "balancer-weighted"

	poolTypeWeighted = "Weighted"
)

const (
	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetScalingFactors    = "getScalingFactors"
)

const (
	graphQLRequestTimeout = 20 * time.Second
)
