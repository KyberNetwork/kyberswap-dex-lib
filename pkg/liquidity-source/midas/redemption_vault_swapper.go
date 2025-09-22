package midas

import (
	"github.com/holiman/uint256"
)

type RedemptionVaultSwapper struct {
	*RedemptionVault
	mTbillRedemptionVault *RedemptionVaultUstb
}

func NewRedemptionVaultSwapper(vaultState *RedemptionVaultWithSwapperState, mTokenDecimals, tokenDecimals uint8) *RedemptionVaultSwapper {
	mTbillRedemptionVault := vaultState.MTbillRedemptionVault
	return &RedemptionVaultSwapper{
		RedemptionVault:       NewRedemptionVault(&vaultState.VaultState, mTokenDecimals, tokenDecimals),
		mTbillRedemptionVault: NewRedemptionVaultUstb(mTbillRedemptionVault, mTokenDecimals, 6),
	}
}

func (v *RedemptionVaultSwapper) RedeemInstant(amountMTokenIn *uint256.Int) (*SwapInfo, error) {
	amountMTokenIn = convertToBase18(amountMTokenIn, v.mTokenDecimals)

	feeAmount, amountMTokenWithoutFee, err := v.calcAndValidateRedeem(amountMTokenIn)
	if err != nil {
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

	if err = v.checkLimits(amountMTokenIn); err != nil {
		return nil, err
	}

	if err = v.checkAllowance(amountTokenOut); err != nil {
		return nil, err
	}

	amountTokenOutWithoutFee = convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals)

	if !v.tokenBalance.Lt(amountTokenOutWithoutFee) {
		// burn
	} else {
		mTbillAmountInBase18 := v.swapMToken1ToMToken2(amountMTokenWithoutFee)

		mTbillAmount := convertFromBase18(mTbillAmountInBase18, v.mTbillRedemptionVault.mTokenDecimals)

		swapInfo, err := v.mTbillRedemptionVault.RedeemInstant(mTbillAmount)
		if err != nil {
			return nil, err
		}

		swapInfo.Gas = redeemInstantSwapperGas

		swapInfo.mTbillAmountInBase18 = mTbillAmountInBase18

		return swapInfo, nil
	}

	return &SwapInfo{
		IsDeposit: false,

		Gas:       redeemInstantDefaultGas,
		Fee:       feeAmount,
		AmountOut: convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals),

		AmountTokenInBase18:  amountTokenOut,
		AmountMTokenInBase18: amountMTokenIn,
	}, nil
}

func (v *RedemptionVaultSwapper) swapMToken1ToMToken2(mToken1Amount *uint256.Int) *uint256.Int {
	amount := truncate(mToken1Amount, v.mTokenDecimals)

	mTokenAmount, _ := new(uint256.Int).MulDivOverflow(amount, v.mTokenRate, v.mTbillRedemptionVault.mTokenRate)

	return mTokenAmount
}

func (v *RedemptionVaultSwapper) UpdateState(swapInfo *SwapInfo) error {
	err := v.RedemptionVault.UpdateState(swapInfo)
	if err != nil {
		return err
	}

	if !v.tokenBalance.Lt(swapInfo.AmountOut) {
		v.tokenBalance = new(uint256.Int).Sub(v.tokenBalance, swapInfo.AmountOut)
	} else {
		if err = v.mTbillRedemptionVault.UpdateState(&SwapInfo{
			IsDeposit: false,
			AmountOut: swapInfo.AmountOut,
		}); err != nil {
			return err
		}
	}

	return nil
}
