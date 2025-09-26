package midas

import (
	"slices"

	"github.com/holiman/uint256"
)

type RedemptionVault struct {
	*ManageableVault

	tokenBalances []*uint256.Int
}

func NewRedemptionVault(vaultState *VaultState, tokenDecimals map[string]uint8) *RedemptionVault {
	return &RedemptionVault{
		ManageableVault: NewManageableVault(vaultState, tokenDecimals),
		tokenBalances:   vaultState.TokenBalances,
	}
}

func (v *RedemptionVault) RedeemInstant(amountMTokenIn *uint256.Int, token string) (*SwapInfo, error) {
	amountMTokenIn = convertToBase18(amountMTokenIn, v.mTokenDecimals)

	feeAmount, amountMTokenWithoutFee, err := v.calcAndValidateRedeem(amountMTokenIn, token)
	if err != nil {
		return nil, err
	}

	if amountMTokenWithoutFee.Sign() == 0 {
		return nil, ErrZeroSwap
	}

	if err = v.checkLimits(amountMTokenIn); err != nil {
		return nil, err
	}

	tokenIndex := v.GetTokenIndex(token)

	amountMTokenInUsd, mTokenRate, err := v.convertTokenToUsd(amountMTokenIn, true, tokenIndex)
	if err != nil {
		return nil, err
	}

	amountTokenOut, tokenOutRate, err := v.convertUsdToToken(amountMTokenInUsd, false, tokenIndex)
	if err != nil {
		return nil, err
	}

	amountTokenOutWithoutFee, _ := new(uint256.Int).MulDivOverflow(amountMTokenWithoutFee, mTokenRate, tokenOutRate)
	amountTokenOutWithoutFee = truncate(amountTokenOutWithoutFee, v.tokenDecimals[token])

	if err = v.checkAllowance(amountTokenOut, tokenIndex); err != nil {
		return nil, err
	}

	amountOut := convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals[token])

	if !v.tokenBalances[tokenIndex].Gt(amountOut) {
		return nil, ErrRVInsufficientBalance
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

func (v *RedemptionVault) UpdateState(swapInfo *SwapInfo, token string) {
	tokenIndex := v.GetTokenIndex(token)
	v.ManageableVault.UpdateState(swapInfo.AmountTokenInBase18, swapInfo.AmountMTokenInBase18, token)
	v.tokenBalances[tokenIndex] = new(uint256.Int).Sub(v.tokenBalances[tokenIndex], swapInfo.amountOut)
}

func (v *RedemptionVault) CloneState() any {
	cloned := *v
	cloned.ManageableVault = v.ManageableVault.CloneState()
	cloned.tokenBalances = slices.Clone(v.tokenBalances)

	return &cloned
}

// feeAmount fee amount in mToken
// amountMTokenWithoutFee amount of mToken without fee
func (v *RedemptionVault) calcAndValidateRedeem(amountMTokenIn *uint256.Int, token string) (*uint256.Int, *uint256.Int, error) {
	tokenIndex := v.GetTokenIndex(token)
	if tokenIndex < 0 {
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

	feeAmount := v.getFeeAmount(amountMTokenIn, 0)
	amountMTokenWithoutFee := new(uint256.Int).Sub(amountMTokenIn, feeAmount)

	return feeAmount, amountMTokenWithoutFee, nil
}
