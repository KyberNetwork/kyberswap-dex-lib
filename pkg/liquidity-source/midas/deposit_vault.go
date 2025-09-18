package midas

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type DepositVault struct {
	*ManageableVault
}

func NewDepositVault(vaultState *VaultState, mTokenDecimals, tokenDecimals uint8, vault common.Address) *DepositVault {
	return &DepositVault{
		ManageableVault: NewVault(vaultState, mTokenDecimals, tokenDecimals, vault),
	}
}

func (v *DepositVault) DepositInstant(tokenAmount *uint256.Int) (*SwapInfo, error) {
	if v.tokenRemoved {
		return nil, ErrTokenRemoved
	}

	if v.paused {
		return nil, ErrDepositVaultPaused
	}

	if v.fnPaused {
		return nil, ErrFnPaused
	}

	tokenAmount = v.convertToBase18(tokenAmount, v.tokenDecimals)

	amountInUsd, tokenInUsdRate, err := v.convertTokenToUsd(tokenAmount, false)
	if err != nil {
		return nil, err
	}

	prevAllowance := v.tokenConfig.Allowance
	if tokenAmount.Gt(prevAllowance) && prevAllowance.Lt(u256.UMax) {
		return nil, ErrMVExceedAllowance
	}

	feeTokenAmount := v.truncate(v.getFeeAmount(tokenAmount), v.tokenDecimals)

	feeInUsd, _ := new(uint256.Int).MulDivOverflow(feeTokenAmount, tokenInUsdRate, u256.BONE)
	mTokenAmount, _, err := v.convertUsdToToken(new(uint256.Int).Sub(amountInUsd, feeInUsd), true)
	if err != nil {
		return nil, err
	}
	return &SwapInfo{
		IsDeposit:          true,
		DepositVault:       &v.vault,
		SwapAmountInBase18: tokenAmount,
		AmountOut:          mTokenAmount,
		Fee:                feeTokenAmount,
		Gas:                depositInstantVaultDefaultGas,
	}, nil
}

func (v *DepositVault) UpdateState(swapInfo *SwapInfo) error {
	v.tokenConfig.Allowance.Sub(v.tokenConfig.Allowance, swapInfo.SwapAmountInBase18)
	v.dailyLimits.Add(v.dailyLimits, v.convertFromBase18(swapInfo.AmountOut, v.tokenDecimals))
	return nil
}
