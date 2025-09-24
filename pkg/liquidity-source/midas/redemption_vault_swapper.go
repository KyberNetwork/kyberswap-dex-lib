package midas

import (
	"github.com/holiman/uint256"
)

type RedemptionVaultSwapper struct {
	*RedemptionVault

	mTbillRedemptionVault IRedemptionVault
	mToken1Balance        *uint256.Int
	mToken2Balance        *uint256.Int
}

func NewRedemptionVaultSwapper(vaultState *VaultState, mTokenDecimals, tokenDecimals uint8) *RedemptionVaultSwapper {
	_, mTbillRedemptionVault, err := newVault(vaultState.MTbillRedemptionVault, vaultState.SwapperVaultType, mTokenDecimals, tokenDecimals)
	if err != nil {
		return nil
	}

	return &RedemptionVaultSwapper{
		RedemptionVault:       NewRedemptionVault(vaultState, mTokenDecimals, tokenDecimals),
		mTbillRedemptionVault: mTbillRedemptionVault,
		mToken1Balance:        vaultState.MToken1Balance,
		mToken2Balance:        vaultState.MToken2Balance,
	}
}

func (v *RedemptionVaultSwapper) RedeemInstant(amountMTokenIn *uint256.Int, tokenOut string) (*SwapInfo, error) {
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

	if v.tokenBalance.Lt(amountTokenOutWithoutFee) {
		mTbillAmountInBase18, err := v.swapMToken1ToMToken2(amountMTokenWithoutFee)
		if err != nil {
			return nil, err
		}

		mTbillAmount := convertFromBase18(mTbillAmountInBase18, 18)

		swapInfo, err := v.mTbillRedemptionVault.RedeemInstant(mTbillAmount, tokenOut)
		if err != nil {
			return nil, err
		}

		swapInfo.gas = redeemInstantSwapperGas
		swapInfo.AmountMTokenInBase18 = amountMTokenIn
		swapInfo.mToken1AmountInBase18 = amountMTokenWithoutFee
		swapInfo.mToken2AmountInBase18 = mTbillAmountInBase18

		return swapInfo, nil
	}

	return &SwapInfo{
		IsDeposit:            false,
		AmountTokenInBase18:  amountTokenOut,
		AmountMTokenInBase18: amountMTokenIn,

		gas:       redeemInstantDefaultGas,
		fee:       feeAmount,
		amountOut: convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals),
	}, nil
}

func (v *RedemptionVaultSwapper) swapMToken1ToMToken2(mToken1Amount *uint256.Int) (*uint256.Int, error) {
	amount := truncate(mToken1Amount, v.mTokenDecimals)

	mTokenAmount, _ := new(uint256.Int).MulDivOverflow(amount, v.mTokenRate, v.mTbillRedemptionVault.GetMTokenRate())

	if v.mToken2Balance.Lt(mTokenAmount) {
		return nil, ErrRVInsufficientMToken2Balance
	}

	return mTokenAmount, nil
}

func (v *RedemptionVaultSwapper) UpdateState(swapInfo *SwapInfo) {
	if !v.tokenBalance.Lt(swapInfo.amountOut) {
		v.RedemptionVault.UpdateState(swapInfo)
	} else {
		v.ManageableVault.UpdateState(swapInfo.AmountTokenInBase18, swapInfo.AmountMTokenInBase18)

		v.mToken1Balance = new(uint256.Int).Add(v.mToken1Balance, swapInfo.mToken1AmountInBase18)
		v.mToken2Balance = new(uint256.Int).Sub(v.mToken2Balance, swapInfo.mToken2AmountInBase18)

		v.mTbillRedemptionVault.UpdateState(&SwapInfo{
			AmountTokenInBase18:  swapInfo.AmountTokenInBase18,
			AmountMTokenInBase18: swapInfo.mToken2AmountInBase18,

			amountOut: swapInfo.amountOut,
		})
	}
}

func (v *RedemptionVaultSwapper) CloneState() any {
	cloned := *v
	cloned.RedemptionVault = v.RedemptionVault.CloneState().(*RedemptionVault)
	cloned.mToken1Balance = new(uint256.Int).Set(v.mToken1Balance)
	cloned.mToken2Balance = new(uint256.Int).Set(v.mToken2Balance)

	return &cloned
}
