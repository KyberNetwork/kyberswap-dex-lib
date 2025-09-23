package midas

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

type RedemptionVaultUstb struct {
	*RedemptionVault
	ustbRedemptionState *RedemptionState
}

func NewRedemptionVaultUstb(vaultState *RedemptionVaultWithUstbState, mTokenDecimals, tokenDecimals uint8) *RedemptionVaultUstb {
	return &RedemptionVaultUstb{
		RedemptionVault:     NewRedemptionVault(&vaultState.VaultState, mTokenDecimals, tokenDecimals),
		ustbRedemptionState: vaultState.UstbRedemptionState,
	}
}

func (v *RedemptionVaultUstb) RedeemInstant(amountMTokenIn *uint256.Int) (*SwapInfo, error) {
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

	if err = v.checkAndRedeemUstb(amountTokenOutWithoutFeeFrom18); err != nil {
		return nil, err
	}

	return &SwapInfo{
		IsDeposit: false,

		Gas:       redeemInstantUstbGas,
		Fee:       feeAmount,
		AmountOut: convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals),

		AmountTokenInBase18:  amountTokenOut,
		AmountMTokenInBase18: amountMTokenIn,
	}, nil
}

func (v *RedemptionVaultUstb) checkAndRedeemUstb(amountTokenOut *uint256.Int) error {
	if !v.tokenBalance.Lt(amountTokenOut) {
		return nil
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
	if v.ustbRedemptionState.UstbBalance.Lt(ustbToRedeem) {
		return ErrRVUInsufficientUstbBalance
	}

	return nil
}

func (v *RedemptionVaultUstb) calculateFee(amount *uint256.Int) *uint256.Int {
	fee, _ := new(uint256.Int).MulDivOverflow(amount, v.ustbRedemptionState.RedemptionFee, feeDenominator)

	return fee
}

func (v *RedemptionVaultUstb) calculateUstbIn(usdcOutAmount *uint256.Int) (*uint256.Int, error) {
	if usdcOutAmount.Sign() == 0 {
		return nil, ErrBadArgsUsdcOutAmountZero
	}

	numerator := new(uint256.Int).Mul(usdcOutAmount, feeDenominator)
	denominator := new(uint256.Int).Sub(feeDenominator, v.ustbRedemptionState.RedemptionFee)
	usdcOutAmountBeforeFee := numerator.Div(numerator, denominator)

	numerator.Mul(usdcOutAmountBeforeFee, v.ustbRedemptionState.ChainLinkFeedPrecision).
		Mul(numerator, v.ustbRedemptionState.SuperstateTokenPrecision)
	denominator.Mul(v.ustbRedemptionState.ChainlinkPrice.Price, usdcPrecision)

	numerator.Add(numerator, denominator).Sub(numerator, u256.U1).Div(numerator, denominator)

	return numerator, nil
}

func (v *RedemptionVaultUstb) UpdateState(swapInfo *SwapInfo) error {
	if !v.tokenBalance.Lt(swapInfo.AmountOut) {
		err := v.RedemptionVault.UpdateState(swapInfo)
		if err != nil {
			return err
		}
	} else {
		err := v.ManageableVault.UpdateState(swapInfo.AmountTokenInBase18, swapInfo.AmountMTokenInBase18)
		if err != nil {
			return err
		}

		missingAmount := new(uint256.Int).Sub(swapInfo.AmountOut, v.tokenBalance)
		v.ustbRedemptionState.UstbBalance = new(uint256.Int).Sub(v.ustbRedemptionState.UstbBalance, missingAmount)

		v.tokenBalance = new(uint256.Int)
	}

	return nil
}
