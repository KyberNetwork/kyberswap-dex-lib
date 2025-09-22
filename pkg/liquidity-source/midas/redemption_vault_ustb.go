package midas

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

type RedemptionVaultUstb struct {
	*RedemptionVault

	chainlinkPrice           *ChainlinkPrice
	redemptionFee            *uint256.Int
	ustbBalance              *uint256.Int
	chainLinkFeedPrecision   *uint256.Int
	superstateTokenPrecision *uint256.Int
}

func NewRedemptionVaultUstb(vaultState *RedemptionVaultWithUstbState, mTokenDecimals, tokenDecimals uint8) *RedemptionVaultUstb {
	return &RedemptionVaultUstb{
		RedemptionVault:          NewRedemptionVault(&vaultState.VaultState, mTokenDecimals, tokenDecimals),
		chainlinkPrice:           vaultState.ChainlinkPrice,
		redemptionFee:            vaultState.RedemptionFee,
		ustbBalance:              vaultState.USTBBalance,
		chainLinkFeedPrecision:   vaultState.ChainLinkFeedPrecision,
		superstateTokenPrecision: vaultState.SuperstateTokenPrecision,
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

	// Skip check tokenOut is USDC

	missingAmount := new(uint256.Int).Sub(amountTokenOut, v.tokenBalance)

	fee := v.calculateFee(missingAmount)
	if fee.Sign() != 0 {
		return ErrRVUUstbFeeNotZero
	}

	ustbToRedeem, err := v.calculateUstbIn(missingAmount)
	if err != nil {
		return err
	}
	if v.ustbBalance.Lt(ustbToRedeem) {
		return ErrRVUInsufficientUstbBalance
	}

	return nil
}

func (v *RedemptionVaultUstb) calculateFee(amount *uint256.Int) *uint256.Int {
	fee, _ := new(uint256.Int).MulDivOverflow(amount, v.redemptionFee, feeDenominator)

	return fee
}

func (v *RedemptionVaultUstb) calculateUstbIn(usdcOutAmount *uint256.Int) (*uint256.Int, error) {
	if usdcOutAmount.Sign() == 0 {
		return nil, ErrBadArgsUsdcOutAmountZero
	}

	numerator := new(uint256.Int).Mul(usdcOutAmount, feeDenominator)
	denominator := new(uint256.Int).Sub(feeDenominator, v.redemptionFee)
	usdcOutAmountBeforeFee := numerator.Div(numerator, denominator)

	numerator.Mul(usdcOutAmountBeforeFee, v.chainLinkFeedPrecision).Mul(numerator, v.superstateTokenPrecision)
	denominator.Mul(v.chainlinkPrice.Price, usdcPrecision)

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
		err := v.ManageableVault.UpdateState(swapInfo)
		if err != nil {
			return err
		}

		missingAmount := new(uint256.Int).Sub(swapInfo.AmountOut, v.tokenBalance)
		v.ustbBalance = new(uint256.Int).Sub(v.ustbBalance, missingAmount)

		v.tokenBalance = new(uint256.Int)
	}

	return nil
}
