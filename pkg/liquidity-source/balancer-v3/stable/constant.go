package stable

import (
	"github.com/holiman/uint256"
)

const (
	DexType = "balancer-v3-stable"

	SubgraphPoolType = "STABLE"

	poolMethodGetAmplificationParameter = "getAmplificationParameter"

	stableSurgeHookMethodGetMaxSurgeFeePercentage    = "getMaxSurgeFeePercentage"
	stableSurgeHookMethodGetSurgeThresholdPercentage = "getSurgeThresholdPercentage"

	baseGas = 237494
)

var (
	// AcceptableMaxSurgeFeePercentage caps max acceptable surge fee to avoid high slippage
	AcceptableMaxSurgeFeePercentage = uint256.NewInt(0.1e18) // 10%
	// AcceptableMaxSurgeFeeByImbalance caps max acceptable surge fee per imbalance to avoid high slippage
	AcceptableMaxSurgeFeeByImbalance = uint256.NewInt(0.156e18) // 0.156% per 1% of imbalance
)
