package midas

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type ManageableVault struct {
	mTokenDecimals uint8
	tokenDecimals  uint8

	tokenRemoved      bool
	paused            bool
	fnPaused          bool
	tokenConfig       *TokenConfig
	instantDailyLimit *uint256.Int
	dailyLimits       *uint256.Int
	minAmount         *uint256.Int
	instantFee        *uint256.Int

	mTokenRate *uint256.Int
	tokenRate  *uint256.Int
}

func NewManageableVault(vaultState *VaultState, mTokenDecimals, tokenDecimals uint8) *ManageableVault {
	return &ManageableVault{
		mTokenDecimals: mTokenDecimals,
		tokenDecimals:  tokenDecimals,

		tokenRemoved:      vaultState.TokenRemoved,
		paused:            vaultState.Paused,
		fnPaused:          vaultState.FnPaused,
		tokenConfig:       vaultState.TokenConfig,
		instantDailyLimit: vaultState.InstantDailyLimit,
		dailyLimits:       vaultState.DailyLimits,
		minAmount:         vaultState.MinAmount,
		instantFee:        vaultState.InstantFee,

		mTokenRate: vaultState.MTokenRate,
		tokenRate:  vaultState.TokenRate,
	}

}

func (v *ManageableVault) getFeeAmount(amount *uint256.Int) *uint256.Int {
	feePercent := new(uint256.Int).Add(v.tokenConfig.Fee, v.instantFee)
	if feePercent.Gt(u256.UBasisPoint) {
		feePercent.Set(u256.UBasisPoint)
	}
	feePercent.MulDivOverflow(amount, feePercent, u256.UBasisPoint)

	return feePercent
}

func (v *ManageableVault) checkAllowance(amount *uint256.Int) error {
	if amount.Gt(v.tokenConfig.Allowance) && v.tokenConfig.Allowance.Lt(u256.UMax) {
		return ErrMVExceedAllowance
	}

	return nil
}

func (v *ManageableVault) checkLimits(amount *uint256.Int) error {
	nextLimitAmount := new(uint256.Int).Add(v.dailyLimits, amount)
	if v.instantDailyLimit.Sign() > 0 && nextLimitAmount.Gt(v.instantDailyLimit) {
		return ErrMVExceedLimit
	}

	return nil
}

func (v *ManageableVault) convertTokenToUsd(amount *uint256.Int, fromMToken bool) (*uint256.Int, *uint256.Int, error) {
	var amountUsd, rate uint256.Int
	if fromMToken {
		rate.Set(v.mTokenRate)
	} else {
		rate.Set(lo.Ternary(v.tokenConfig.Stable, StableCoinRate, v.tokenRate))
	}

	if rate.Sign() == 0 {
		return nil, nil, ErrRateZero
	}

	amountUsd.MulDivOverflow(amount, &rate, u256.BONE)

	return &amountUsd, &rate, nil
}

func (v *ManageableVault) convertUsdToToken(amountUsd *uint256.Int, toMToken bool) (*uint256.Int, *uint256.Int, error) {
	var amount, rate uint256.Int
	if toMToken {
		rate.Set(v.mTokenRate)
	} else {
		rate.Set(lo.Ternary(v.tokenConfig.Stable, StableCoinRate, v.tokenRate))
	}

	if rate.Sign() == 0 {
		return nil, nil, ErrRateZero
	}

	amount.MulDivOverflow(amountUsd, u256.BONE, &rate)

	return &amount, &rate, nil
}

func truncate(value *uint256.Int, decimals uint8) *uint256.Int {
	if value.Sign() == 0 || decimals == 18 {
		return value
	}

	diff := 18 - decimals
	if diff > 0 {
		value.Div(value, u256.TenPow(diff)).Mul(value, u256.TenPow(diff))
	} else {
		value.Mul(value, u256.TenPow(-diff)).Div(value, u256.TenPow(-diff))
	}

	return value
}

func convertToBase18(amount *uint256.Int, decimals uint8) *uint256.Int {
	if amount == nil {
		return new(uint256.Int)
	}

	if amount.Sign() == 0 || decimals == 18 {
		return new(uint256.Int).Set(amount)
	}

	diff := 18 - decimals
	if diff > 0 {
		return new(uint256.Int).Mul(amount, u256.TenPow(diff))
	}

	return new(uint256.Int).Div(amount, u256.TenPow(-diff))
}

func convertFromBase18(amount *uint256.Int, decimals uint8) *uint256.Int {
	if amount == nil {
		return new(uint256.Int)
	}

	if amount.Sign() == 0 || decimals == 18 {
		return new(uint256.Int).Set(amount)
	}

	diff := 18 - decimals
	if diff > 0 {
		return new(uint256.Int).Div(amount, u256.TenPow(diff))
	}

	return new(uint256.Int).Mul(amount, u256.TenPow(-diff))
}
