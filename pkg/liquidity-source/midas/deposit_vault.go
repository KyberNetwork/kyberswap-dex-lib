package midas

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type DepositVault struct {
	*ManageableVault

	minMTokenAmountForFirstDeposit *uint256.Int
	totalMinted                    *uint256.Int
	mTokenTotalSuply               *uint256.Int
	maxSupplyCap                   *uint256.Int
}

func NewDepositVault(vaultState *VaultState, mTokenDecimals, tokenDecimals uint8) *DepositVault {
	if vaultState == nil {
		return nil
	}

	return &DepositVault{
		ManageableVault:                NewManageableVault(vaultState, mTokenDecimals, tokenDecimals),
		minMTokenAmountForFirstDeposit: vaultState.MinMTokenAmountForFirstDeposit,
		totalMinted:                    vaultState.TotalMinted,
		mTokenTotalSuply:               vaultState.MTokenTotalSupply,
		maxSupplyCap:                   vaultState.MaxSupplyCap,
	}
}

func (v *DepositVault) DepositInstant(amountToken *uint256.Int) (*SwapInfo, error) {
	amountToken = convertToBase18(amountToken, v.tokenDecimals)

	feeAmount, mintAmount, err := v.calcAndValidateDeposit(amountToken)
	if err != nil {
		return nil, err
	}

	if err = v.checkLimits(mintAmount); err != nil {
		return nil, err
	}

	if v.maxSupplyCap != nil && v.maxSupplyCap.Sign() > 0 {
		if new(uint256.Int).Add(v.mTokenTotalSuply, mintAmount).Gt(v.maxSupplyCap) {
			return nil, ErrDvMaxSupplyCapExceeded
		}
	}

	return &SwapInfo{
		IsDeposit:            true,
		AmountTokenInBase18:  amountToken,
		AmountMTokenInBase18: mintAmount,

		gas:       depositInstantDefaultGas,
		fee:       feeAmount,
		amountOut: convertFromBase18(mintAmount, v.mTokenDecimals),
	}, nil
}

func (v *DepositVault) UpdateState(swapInfo *SwapInfo) {
	v.ManageableVault.UpdateState(swapInfo.AmountTokenInBase18, swapInfo.AmountMTokenInBase18)

	v.totalMinted = new(uint256.Int).Add(v.totalMinted, swapInfo.AmountMTokenInBase18)
}

func (v *DepositVault) CloneState() any {
	cloned := *v
	cloned.ManageableVault = v.ManageableVault.CloneState()
	cloned.totalMinted = new(uint256.Int).Set(v.totalMinted)

	return &cloned
}

// feeTokenAmount fee amount in tokenIn
// mTokenAmount mToken amount for mint
func (v *DepositVault) calcAndValidateDeposit(amountToken *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if v.tokenRemoved {
		return nil, nil, ErrTokenRemoved
	}

	if v.paused {
		return nil, nil, ErrDVPaused
	}

	if v.fnPaused {
		return nil, nil, ErrDepositInstantFnPaused
	}

	amountInUsd, tokenInUsdRate, err := v.convertTokenToUsd(amountToken, false)
	if err != nil {
		return nil, nil, err
	}

	if err = v.checkAllowance(amountToken); err != nil {
		return nil, nil, err
	}

	feeTokenAmount := truncate(v.getFeeAmount(amountToken), v.tokenDecimals)

	feeInUsd, _ := new(uint256.Int).MulDivOverflow(feeTokenAmount, tokenInUsdRate, u256.BONE)

	mTokenAmount, _, err := v.convertUsdToToken(new(uint256.Int).Sub(amountInUsd, feeInUsd), true)
	if err != nil {
		return nil, nil, err
	}

	if mTokenAmount.Sign() == 0 {
		return nil, nil, ErrDVInvalidMintAmount
	}

	if mTokenAmount.Lt(v.minAmount) {
		return nil, nil, ErrDvMintAmountLtMin
	}

	if v.totalMinted.Sign() == 0 && mTokenAmount.Lt(v.minMTokenAmountForFirstDeposit) {
		return nil, nil, ErrDvMTokenAmountLtMin
	}

	return feeTokenAmount, mTokenAmount, nil
}
