package vault

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

func (v *Vault) callComputeDynamicSwapFeeHook(poolSwapParams shared.PoolSwapParams) (*uint256.Int, error) {
	success, swapFeePercentage, err := v.hook.OnComputeDynamicSwapFeePercentage(poolSwapParams)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, ErrDynamicSwapFeeHookFailed
	}

	if swapFeePercentage.Gt(MAX_FEE_PERCENTAGE) {
		return nil, ErrDynamicSwapFeeHookFailed
	}

	return swapFeePercentage, nil
}

// (bool success, uint256 swapFeePercentage) = hooksContract.onComputeDynamicSwapFeePercentage(
// 	swapParams,
// 	pool,
// 	staticSwapFeePercentage
// );

// if (success == false) {
// 	revert IVaultErrors.DynamicSwapFeeHookFailed();
// }

// // A 100% fee is not supported. In the ExactOut case, the Vault divides by the complement of the swap fee.
// // The minimum precision constraint provides an additional buffer.
// if (swapFeePercentage > MAX_FEE_PERCENTAGE) {
// 	revert IVaultErrors.PercentageAboveMax();
// }

// return swapFeePercentage;
