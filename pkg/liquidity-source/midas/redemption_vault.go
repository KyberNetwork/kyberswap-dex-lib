package midas

import (
	"github.com/holiman/uint256"
)

type RedemptionVault struct {
	*ManageableVault
}

func NewRedemptionVault(vaultState *VaultState, mTokenDecimals, tokenDecimals uint8) *RedemptionVault {
	return &RedemptionVault{
		ManageableVault: NewManageableVault(vaultState, mTokenDecimals, tokenDecimals),
	}
}

func (v *RedemptionVault) RedeemInstant(amountMTokenIn *uint256.Int) (*SwapInfo, error) {
	amountMTokenIn = convertToBase18(amountMTokenIn, v.mTokenDecimals)

	feeAmount, amountMTokenWithoutFee, err := v.calcAndValidateRedeem(amountMTokenIn)
	if err != nil {
		return nil, err
	}

	if err = v.checkLimits(amountMTokenIn); err != nil {
		return nil, err
	}

	amountMTokenInUsd, mTokenRate, err := v.convertTokenToUsd(amountMTokenIn, true)
	if err != nil {
		return nil, err
	}

	amountTokenOut, tokenOutRate, err := v.convertUsdToToken(amountMTokenInUsd, false)
	if err != nil {
		return nil, err
	}

	amountTokenOutWithoutFee, _ := new(uint256.Int).MulDivOverflow(amountMTokenWithoutFee, mTokenRate, tokenOutRate)
	amountTokenOutWithoutFee = truncate(amountTokenOutWithoutFee, v.tokenDecimals)

	if err = v.checkAllowance(amountTokenOut); err != nil {
		return nil, err
	}

	return &SwapInfo{
		IsDeposit:          false,
		SwapAmountInBase18: amountMTokenIn,

		Gas:       redeemInstantDefaultGas,
		Fee:       feeAmount,
		AmountOut: convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals),
	}, nil
}

func (v *RedemptionVault) UpdateState(swapInfo *SwapInfo) error {
	v.tokenConfig.Allowance.Sub(v.tokenConfig.Allowance, swapInfo.SwapAmountInBase18)

	v.dailyLimits.Add(v.dailyLimits, swapInfo.AmountOut)

	return nil
}

// feeAmount fee amount in mToken
// amountMTokenWithoutFee amount of mToken without fee
func (v *RedemptionVault) calcAndValidateRedeem(amountMTokenIn *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if v.tokenRemoved {
		return nil, nil, ErrTokenRemoved
	}

	if v.paused {
		return nil, nil, ErrRedemptionVaultPaused
	}

	if v.fnPaused {
		return nil, nil, ErrRedeemInstantFnPaused
	}

	if amountMTokenIn.Sign() == 0 {
		return nil, nil, ErrInvalidAmount
	}

	feeAmount := v.getFeeAmount(amountMTokenIn)
	amountMTokenWithoutFee := new(uint256.Int).Sub(amountMTokenIn, feeAmount)

	return feeAmount, amountMTokenWithoutFee, nil
}
