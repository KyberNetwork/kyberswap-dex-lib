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

func NewRedemptionVaultUstb(vaultState *VaultState, mTokenDecimals, tokenDecimals uint8) *RedemptionVaultUstb {
	return &RedemptionVaultUstb{
		RedemptionVault: NewRedemptionVault(vaultState, mTokenDecimals, tokenDecimals),
		ustbRedemption:  vaultState.Redemption,
	}
}

func (v *RedemptionVaultUstb) RedeemInstant(amountMTokenIn *uint256.Int, tokenOut string) (*SwapInfo, error) {
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

	if err = v.checkAllowance(amountTokenOut); err != nil {
		return nil, err
	}

	amountTokenOutWithoutFeeFrom18, _ := new(uint256.Int).MulDivOverflow(amountMTokenWithoutFee, mTokenRate, tokenOutRate)
	amountTokenOutWithoutFeeFrom18 = convertFromBase18(amountTokenOutWithoutFeeFrom18, v.tokenDecimals)
	amountTokenOutWithoutFee := convertToBase18(amountTokenOutWithoutFeeFrom18, v.tokenDecimals)

	if err = v.checkAndRedeemUstb(tokenOut, amountTokenOutWithoutFeeFrom18); err != nil {
		return nil, err
	}

	return &SwapInfo{
		IsDeposit:            false,
		AmountTokenInBase18:  amountTokenOut,
		AmountMTokenInBase18: amountMTokenIn,

		gas:       redeemInstantUstbGas,
		fee:       feeAmount,
		amountOut: convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals),
	}, nil
}

func (v *RedemptionVaultUstb) checkAndRedeemUstb(tokenOut string, amountTokenOut *uint256.Int) error {
	if !v.tokenBalance.Lt(amountTokenOut) {
		return nil
	}

	if !strings.EqualFold(tokenOut, v.ustbRedemption.Usdc.String()) {
		return ErrRVUInvalidToken
	}

	missingAmount := new(uint256.Int).Sub(amountTokenOut, v.tokenBalance)

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

func (v *RedemptionVaultUstb) UpdateState(swapInfo *SwapInfo) {
	if !v.tokenBalance.Lt(swapInfo.amountOut) {
		v.RedemptionVault.UpdateState(swapInfo)
	} else {
		v.ManageableVault.UpdateState(swapInfo.AmountTokenInBase18, swapInfo.AmountMTokenInBase18)

		missingAmount := new(uint256.Int).Sub(swapInfo.amountOut, v.tokenBalance)
		v.ustbRedemption.UstbBalance = new(uint256.Int).Sub(v.ustbRedemption.UstbBalance, missingAmount)

		v.tokenBalance = new(uint256.Int)
	}
}

func (v *RedemptionVaultUstb) CloneState() any {
	cloned := *v
	cloned.RedemptionVault = v.RedemptionVault.CloneState().(*RedemptionVault)

	return &cloned
}
