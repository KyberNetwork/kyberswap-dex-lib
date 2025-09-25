package midas

import (
	"github.com/holiman/uint256"
)

type RedemptionVault struct {
	*ManageableVault

	tokenBalance *uint256.Int
}

func NewRedemptionVault(vaultState *VaultState, mTokenDecimals, tokenDecimals uint8) *RedemptionVault {
	return &RedemptionVault{
		ManageableVault: NewManageableVault(vaultState, mTokenDecimals, tokenDecimals),
		tokenBalance:    vaultState.TokenBalance,
	}
}

func (v *RedemptionVault) RedeemInstant(amountMTokenIn *uint256.Int, _ string) (*SwapInfo, error) {
	amountMTokenIn = convertToBase18(amountMTokenIn, v.mTokenDecimals)

	feeAmount, amountMTokenWithoutFee, err := v.calcAndValidateRedeem(amountMTokenIn)
	if err != nil {
		return nil, err
	}

	if amountMTokenWithoutFee.Sign() == 0 {
		return nil, ErrInvalidSwap
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

	amountOut := convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals)

	if !v.tokenBalance.Gt(amountOut) {
		return nil, ErrDVInsufficientBalance
	}

	return &SwapInfo{
		IsDeposit:            false,
		AmountTokenInBase18:  amountTokenOut,
		AmountMTokenInBase18: amountMTokenIn,

		gas:       redeemInstantDefaultGas,
		fee:       feeAmount,
		amountOut: amountOut,
	}, nil
}

func (v *RedemptionVault) UpdateState(swapInfo *SwapInfo) {
	v.ManageableVault.UpdateState(swapInfo.AmountTokenInBase18, swapInfo.AmountMTokenInBase18)
	v.tokenBalance = new(uint256.Int).Sub(v.tokenBalance, swapInfo.amountOut)
}

func (v *RedemptionVault) CloneState() any {
	cloned := *v
	cloned.ManageableVault = v.ManageableVault.CloneState()
	cloned.tokenBalance = new(uint256.Int).Set(v.tokenBalance)

	return &cloned
}

// feeAmount fee amount in mToken
// amountMTokenWithoutFee amount of mToken without fee
func (v *RedemptionVault) calcAndValidateRedeem(amountMTokenIn *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if v.tokenRemoved {
		return nil, nil, ErrTokenRemoved
	}

	if v.paused {
		return nil, nil, ErrRVPaused
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
