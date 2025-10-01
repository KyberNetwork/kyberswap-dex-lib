package midas

import (
	"strings"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type RedemptionVaultUstb struct {
	*RedemptionVault

	ustbRedemption *RedemptionState
}

func NewRedemptionVaultUstb(vaultState *VaultState, tokenDecimals map[string]uint8) *RedemptionVaultUstb {
	return &RedemptionVaultUstb{
		RedemptionVault: NewRedemptionVault(vaultState, tokenDecimals),
		ustbRedemption:  vaultState.Redemption,
	}
}

func (v *RedemptionVaultUstb) RedeemInstant(amountMTokenIn *uint256.Int, token string) (*SwapInfo, error) {
	amountMTokenIn = convertToBase18(amountMTokenIn, v.mTokenDecimals)

	feeAmount, amountMTokenWithoutFee, err := v.calcAndValidateRedeem(amountMTokenIn, token)
	if err != nil {
		return nil, err
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

	if err = v.checkAllowance(amountTokenOut, tokenIndex); err != nil {
		return nil, err
	}

	amountTokenOutWithoutFeeFrom18, _ := new(uint256.Int).MulDivOverflow(amountMTokenWithoutFee, mTokenRate, tokenOutRate)
	amountTokenOutWithoutFeeFrom18 = convertFromBase18(amountTokenOutWithoutFeeFrom18, v.tokenDecimals[token])
	amountTokenOutWithoutFee := convertToBase18(amountTokenOutWithoutFeeFrom18, v.tokenDecimals[token])

	if err = v.checkAndRedeemUstb(amountTokenOutWithoutFeeFrom18, tokenIndex); err != nil {
		return nil, err
	}

	return &SwapInfo{
		IsDeposit:            false,
		amountTokenInBase18:  amountTokenOut,
		amountMTokenInBase18: amountMTokenIn,

		gas:       redeemInstantUstbGas,
		fee:       feeAmount,
		amountOut: convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals[token]),
	}, nil
}

func (v *RedemptionVaultUstb) checkAndRedeemUstb(amountTokenOut *uint256.Int, tokenIndex int) error {
	if !v.tokenBalances[tokenIndex].Lt(amountTokenOut) {
		return nil
	}

	if !strings.EqualFold(v.paymentTokens[tokenIndex], v.ustbRedemption.Usdc.String()) {
		return ErrRVUInvalidToken
	}

	missingAmount := new(uint256.Int).Sub(amountTokenOut, v.tokenBalances[tokenIndex])

	fee := v.calculateFee(missingAmount)
	if fee.Sign() != 0 {
		return ErrRVUUstbFeeNotZero
	}

	ustbToRedeem, err := v.calculateUstbIn(missingAmount)
	if err != nil {
		return err
	}
	if v.ustbRedemption.UstbBalance.Lt(ustbToRedeem) {
		return ErrRVUInsufficientUstbBalance
	}

	return nil
}

func (v *RedemptionVaultUstb) calculateFee(amount *uint256.Int) *uint256.Int {
	fee, _ := new(uint256.Int).MulDivOverflow(amount, v.ustbRedemption.RedemptionFee, feeDenominator)

	return fee
}

func (v *RedemptionVaultUstb) calculateUstbIn(usdcOutAmount *uint256.Int) (*uint256.Int, error) {
	if usdcOutAmount.Sign() == 0 {
		return nil, ErrBadArgsUsdcOutAmountZero
	}

	numerator := new(uint256.Int).Mul(usdcOutAmount, feeDenominator)
	denominator := new(uint256.Int).Sub(feeDenominator, v.ustbRedemption.RedemptionFee)
	usdcOutAmountBeforeFee := numerator.Div(numerator, denominator)

	numerator.Mul(usdcOutAmountBeforeFee, v.ustbRedemption.ChainLinkFeedPrecision).
		Mul(numerator, v.ustbRedemption.SuperstateTokenPrecision)
	denominator.Mul(v.ustbRedemption.ChainlinkPrice.Price, usdcPrecision)

	numerator.Add(numerator, denominator).Sub(numerator, u256.U1).Div(numerator, denominator)

	return numerator, nil
}

func (v *RedemptionVaultUstb) UpdateState(swapInfo *SwapInfo, token string) {
	tokenIndex := v.GetTokenIndex(token)
	if !v.tokenBalances[tokenIndex].Lt(swapInfo.amountOut) {
		v.RedemptionVault.UpdateState(swapInfo, token)
	} else {
		v.ManageableVault.UpdateState(swapInfo.amountTokenInBase18, swapInfo.amountMTokenInBase18, token)

		missingAmount := new(uint256.Int).Sub(swapInfo.amountOut, v.tokenBalances[tokenIndex])
		v.ustbRedemption.UstbBalance = new(uint256.Int).Sub(v.ustbRedemption.UstbBalance, missingAmount)

		v.tokenBalances[tokenIndex] = new(uint256.Int)
	}
}

func (v *RedemptionVaultUstb) CloneState() any {
	cloned := *v
	cloned.RedemptionVault = v.RedemptionVault.CloneState().(*RedemptionVault)

	return &cloned
}
