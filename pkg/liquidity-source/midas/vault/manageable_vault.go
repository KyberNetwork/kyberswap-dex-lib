package midas

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/midas"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type ManageableVault struct {
	tokenDecimals  uint8
	mTokenDecimals uint8

	tokenRemoved           bool
	tokenConfig            *midas.TokenConfig
	depositInstantFnPaused bool
	redeemInstantFnPaused  bool
	dailyLimits            *uint256.Int
	mTokenRate             *uint256.Int
	tokenRate              *uint256.Int
	minAmount              *uint256.Int
	instantFee             *uint256.Int
	instantDailyLimit      *uint256.Int

	vaultPaused bool
	vault       common.Address
}

func (v *ManageableVault) convertTokenToUsd(amount *uint256.Int, isMToken bool) (*uint256.Int, *uint256.Int, error) {
	var amountUsd, rate uint256.Int
	if isMToken {
		rate.Set(v.mTokenRate)
	} else {
		rate.Set(lo.Ternary(v.tokenConfig.Stable, midas.StableCoinRate, v.tokenRate))
	}

	if rate.Sign() == 0 {
		return nil, nil, midas.ErrRateZero
	}

	amountUsd.MulDivOverflow(amount, &rate, u256.BONE)

	return &amountUsd, &rate, nil
}

func (v *ManageableVault) convertUsdToToken(amountUsd *uint256.Int, isMToken bool) (*uint256.Int, *uint256.Int, error) {
	var amount, rate uint256.Int
	if isMToken {
		rate.Set(v.mTokenRate)
	} else {
		rate.Set(lo.Ternary(v.tokenConfig.Stable, midas.StableCoinRate, v.tokenRate))
	}

	if rate.Sign() == 0 {
		return nil, nil, midas.ErrRateZero
	}

	amount.MulDivOverflow(amountUsd, u256.BONE, &rate)

	return &amount, &rate, nil
}

func (v *ManageableVault) getFeeAmount(amount *uint256.Int) *uint256.Int {
	feePercent := new(uint256.Int).Add(v.tokenConfig.Fee, v.instantFee)
	if feePercent.Gt(u256.UBasisPoint) {
		feePercent.Set(u256.UBasisPoint)
	}
	feePercent.MulDivOverflow(amount, feePercent, u256.UBasisPoint)

	return feePercent
}

func (v *ManageableVault) truncate(value *uint256.Int, decimals uint8) *uint256.Int {
	if value.Sign() == 0 || decimals == 18 {
		return value
	}

	diff := 18 - decimals
	if diff > 0 {
		value.Div(value, u256.TenPow(diff)).Mul(value, u256.TenPow(diff))
	}

	return value.Mul(value, u256.TenPow(-diff)).Div(value, u256.TenPow(-diff))
}

func (v *ManageableVault) convertToBase18(amount *uint256.Int, decimals uint8) *uint256.Int {
	if amount.Sign() == 0 || decimals == 18 {
		return new(uint256.Int).Set(amount)
	}

	diff := 18 - decimals
	if diff > 0 {
		return new(uint256.Int).Mul(amount, u256.TenPow(diff))
	}

	return new(uint256.Int).Div(amount, u256.TenPow(-diff))
}
