package stable

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
	AcceptableMaxSurgeFeeByImbalance = uint256.NewInt(0.1e18) // 0.1% per 1% of imbalance

	stablesByChain = map[valueobject.ChainID]map[string]bool{
		valueobject.ChainIDBase: {
			"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": true,
		},
		valueobject.ChainIDSonic: {
			"0x29219dd400f2bf60e5a23d13be72b486d4038894": true,
		},
	}
)
