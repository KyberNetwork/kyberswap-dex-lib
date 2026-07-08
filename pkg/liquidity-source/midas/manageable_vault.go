package midas

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type ManageableVault struct {
	mToken        string
	tokenDecimals map[string]uint8

	paymentTokens        []string
	paused               bool
	fnPaused             bool
	tokenConfigs         []TokenConfig
	instantDailyLimit    *uint256.Int
	dailyLimits          *uint256.Int
	minAmount            *uint256.Int
	instantFee           *uint256.Int
	waivedFeeRestriction bool
	mTokenDecimals       uint8

	mTokenRate *uint256.Int
	tokenRates []*uint256.Int
}

func NewManageableVault(vaultState *VaultState, tokenDecimals map[string]uint8) *ManageableVault {
	return &ManageableVault{
		mToken:        vaultState.MToken,
		tokenDecimals: tokenDecimals,

		paymentTokens:        vaultState.PaymentTokens,
		paused:               vaultState.Paused,
		fnPaused:             vaultState.FnPaused,
		tokenConfigs:         vaultState.TokenConfigs,
		instantDailyLimit:    vaultState.InstantDailyLimit,
		dailyLimits:          vaultState.DailyLimits,
		minAmount:            vaultState.MinAmount,
		instantFee:           vaultState.InstantFee,
		waivedFeeRestriction: vaultState.WaivedFeeRestriction,
		mTokenDecimals:       vaultState.MTokenDecimals,

		mTokenRate: vaultState.MTokenRate,
		tokenRates: vaultState.TokenRates,
	}
}

func (v *ManageableVault) GetMTokenRate() *uint256.Int {
	return v.mTokenRate
}

func (v *ManageableVault) GetMToken() string {
	return v.mToken
}

func (v *ManageableVault) getFeeAmount(amount *uint256.Int, tokenIndex int) *uint256.Int {
	if v.waivedFeeRestriction {
		return new(uint256.Int)
	}

	tokenConfig := v.tokenConfigs[tokenIndex]

	feePercent := new(uint256.Int).Add(tokenConfig.Fee, v.instantFee)
	if feePercent.Gt(u256.UBasisPoint) {
		feePercent.Set(u256.UBasisPoint)
	}
	feePercent.MulDivOverflow(amount, feePercent, u256.UBasisPoint)

	return feePercent
}

func (v *ManageableVault) checkAllowance(tokenAmount *uint256.Int, tokenIndex int) error {
	tokenConfig := v.tokenConfigs[tokenIndex]
	if tokenAmount.Gt(tokenConfig.Allowance) && tokenConfig.Allowance.Lt(u256.UMax) {
		return ErrMVExceedAllowance
	}

	return nil
}

func (v *ManageableVault) checkLimits(mTokenAmount *uint256.Int) error {
	nextLimitAmount := new(uint256.Int).Add(v.dailyLimits, mTokenAmount)
	if nextLimitAmount.Gt(v.instantDailyLimit) {
		return ErrMVExceedLimit
	}

	return nil
}

func (v *ManageableVault) UpdateState(amountTokenInBase18, amountMTokenInBase18 *uint256.Int, token string) {
	tokenIndex := v.GetTokenIndex(token)
	v.tokenConfigs[tokenIndex].Allowance = new(uint256.Int).Sub(v.tokenConfigs[tokenIndex].Allowance, amountTokenInBase18)
	v.dailyLimits = new(uint256.Int).Add(v.dailyLimits, amountMTokenInBase18)
}

func (v *ManageableVault) CloneState() *ManageableVault {
	cloned := *v
	cloned.tokenConfigs = lo.Map(v.tokenConfigs, func(cfg TokenConfig, _ int) TokenConfig {
		cfg.Allowance = new(uint256.Int).Set(cfg.Allowance)
		return cfg
	})
	cloned.dailyLimits = new(uint256.Int).Set(cloned.dailyLimits)

	return &cloned
}

func (v *ManageableVault) convertTokenToUsd(amount *uint256.Int, fromMToken bool, tokenIndex int) (*uint256.Int, *uint256.Int, error) {
	var amountUsd, rate uint256.Int
	if fromMToken {
		rate.Set(v.mTokenRate)
	} else {
		_rate := lo.Ternary(v.tokenConfigs[tokenIndex].Stable, stableCoinRate, v.tokenRates[tokenIndex])
		if _rate == nil {
			return nil, nil, ErrInvalidTokenRate
		}
		rate.Set(_rate)
	}

	if rate.Sign() == 0 {
		return nil, nil, ErrRateZero
	}

	amountUsd.MulDivOverflow(amount, &rate, u256.BONE)

	return &amountUsd, &rate, nil
}

func (v *ManageableVault) convertUsdToToken(amountUsd *uint256.Int, toMToken bool, tokenIndex int) (*uint256.Int, *uint256.Int, error) {
	var amount, rate uint256.Int
	if toMToken {
		rate.Set(v.mTokenRate)
	} else {
		_rate := lo.Ternary(v.tokenConfigs[tokenIndex].Stable, stableCoinRate, v.tokenRates[tokenIndex])
		if _rate == nil {
			return nil, nil, ErrInvalidTokenRate
		}
		rate.Set(_rate)
	}

	if rate.Sign() == 0 {
		return nil, nil, ErrRateZero
	}

	amount.MulDivOverflow(amountUsd, u256.BONE, &rate)

	return &amount, &rate, nil
}

func (v *ManageableVault) GetTokenIndex(address string) int {
	for i, token := range v.paymentTokens {
		if token == address {
			return i
		}
	}

	return -1
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
