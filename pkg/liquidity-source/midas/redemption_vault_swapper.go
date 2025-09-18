package midas

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type RedemptionVaultSwapper struct {
	*ManageableVault
	tokenBalance *uint256.Int
	mTbillRate   *uint256.Int
}

func NewRedemptionVaultSwapper(vaultState *RedemptionVaultState, mTokenDecimals, tokenDecimals uint8,
	vault common.Address) *RedemptionVaultSwapper {
	return &RedemptionVaultSwapper{
		ManageableVault: NewVault(&vaultState.VaultState, mTokenDecimals, tokenDecimals, vault),
		tokenBalance:    vaultState.TokenBalance,
		mTbillRate:      vaultState.MTbillRate,
	}
}

func (v *RedemptionVaultSwapper) RedeemInstant(amountMTokenIn *uint256.Int) (*SwapInfo, error) {
	if v.tokenRemoved {
		return nil, ErrTokenRemoved
	}

	if v.paused {
		return nil, ErrRedemptionVaultPaused
	}

	if v.fnPaused {
		return nil, ErrFnPaused
	}

	amountMTokenIn = v.convertToBase18(amountMTokenIn, v.mTokenDecimals)

	feeAmount := v.getFeeAmount(amountMTokenIn)

	if !amountMTokenIn.Gt(feeAmount) {
		return nil, ErrRVAmountMTokenLteFee
	}

	amountMTokenWithoutFee := new(uint256.Int).Sub(amountMTokenIn, feeAmount)

	amountMTokenInUsd, mTokenRate, err := v.convertTokenToUsd(amountMTokenIn, true)
	if err != nil {
		return nil, err
	}

	amountTokenOut, tokenOutRate, err := v.convertUsdToToken(amountMTokenInUsd, false)
	if err != nil {
		return nil, err
	}

	amountTokenOutWithoutFee, _ := new(uint256.Int).MulDivOverflow(amountMTokenWithoutFee, mTokenRate, tokenOutRate)
	amountTokenOutWithoutFee = v.truncate(amountTokenOutWithoutFee, v.tokenDecimals)

	nextLimitAmount := new(uint256.Int).Add(v.dailyLimits, amountMTokenIn)
	if v.instantDailyLimit.Sign() > 0 && nextLimitAmount.Gt(v.instantDailyLimit) {
		return nil, ErrMVExceedLimit
	}

	prevAllowance := v.tokenConfig.Allowance
	if amountTokenOut.Gt(prevAllowance) && prevAllowance.Lt(u256.UMax) {
		return nil, ErrMVExceedAllowance
	}

	if !v.tokenBalance.Lt(v.convertFromBase18(amountTokenOutWithoutFee, v.tokenDecimals)) {
		// burn
	} else {
		mTbillAmount := v.swapMToken1ToMToken2(amountTokenOutWithoutFee)

		// TODO: increase allowance for mTbill vault

	}

	return nil, nil
}

func (v *RedemptionVaultSwapper) swapMToken1ToMToken2(mToken1Amount *uint256.Int) *uint256.Int {
	mTokenAmount, _ := new(uint256.Int).MulDivOverflow(mToken1Amount, v.mTokenRate, v.mTbillRate)

	return mTokenAmount
}

func (v *RedemptionVaultSwapper) UpdateState(_ *SwapInfo) error {
	return nil
}
