package vault

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

func (v *Vault) callBeforeSwapHook(poolSwapParams shared.PoolSwapParams) error {
	success, err := v.hook.OnBeforeSwap(poolSwapParams)
	if err != nil {
		return err
	}

	if !success {
		return ErrBeforeSwapHookFailed
	}

	return nil
}

func (v *Vault) callAfterSwapHook(
	vaultParamSwap shared.VaultSwapParams,
	amountGivenScaled18,
	amountCalculatedScaled18, amountCalculatedRaw *uint256.Int,
) (*uint256.Int, error) {
	amountInScaled18, amountOutScaled18 := amountCalculatedScaled18, amountGivenScaled18
	if vaultParamSwap.Kind == shared.EXACT_IN {
		amountInScaled18, amountOutScaled18 = amountGivenScaled18, amountCalculatedScaled18
	}

	success, hookAdjustedAmountCalculatedRaw, err := v.hook.OnAfterSwap(shared.AfterSwapParams{
		Kind:                     vaultParamSwap.Kind,
		IndexIn:                  vaultParamSwap.IndexIn,
		IndexOut:                 vaultParamSwap.IndexOut,
		AmountInScaled18:         amountInScaled18,
		AmountOutScaled18:        amountOutScaled18,
		TokenInBalanceScaled18:   v.balancesLiveScaled18[vaultParamSwap.IndexIn],
		TokenOutBalanceScaled18:  v.balancesLiveScaled18[vaultParamSwap.IndexOut],
		AmountCalculatedScaled18: amountCalculatedScaled18,
		AmountCalculatedRaw:      amountCalculatedRaw,
	})
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, ErrAfterSwapHookFailed
	}

	if !v.hooksConfig.EnableHookAdjustedAmounts {
		return amountCalculatedRaw, nil
	}

	return hookAdjustedAmountCalculatedRaw, nil
}

func (v *Vault) callComputeDynamicSwapFeeHook(poolSwapParams shared.PoolSwapParams) (*uint256.Int, error) {
	success, swapFeePercentage, err := v.hook.OnComputeDynamicSwapFeePercentage(poolSwapParams)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, ErrDynamicSwapFeeHookFailed
	}

	if swapFeePercentage.Gt(MaxFeePercentage) {
		return nil, ErrDynamicSwapFeeHookFailed
	}

	return swapFeePercentage, nil
}
