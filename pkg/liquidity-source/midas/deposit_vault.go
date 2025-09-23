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

	if new(uint256.Int).Add(v.mTokenTotalSuply, mintAmount).Gt(v.maxSupplyCap) {
		return nil, ErrDvMaxSupplyCapExceeded
	}

	return &SwapInfo{
		IsDeposit: true,

		Gas:       depositInstantDefaultGas,
		Fee:       feeAmount,
		AmountOut: convertFromBase18(mintAmount, v.mTokenDecimals),

		AmountTokenInBase18:  amountToken,
		AmountMTokenInBase18: mintAmount,
	}, nil
}

func (v *DepositVault) UpdateState(swapInfo *SwapInfo) error {
	v.tokenConfig.Allowance = new(uint256.Int).Sub(v.tokenConfig.Allowance, swapInfo.AmountTokenInBase18)

	v.dailyLimits = new(uint256.Int).Add(v.dailyLimits, swapInfo.AmountMTokenInBase18)

	return nil
}

// feeTokenAmount fee amount in tokenIn
// mTokenAmount mToken amount for mint
func (v *DepositVault) calcAndValidateDeposit(amountToken *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if v.tokenRemoved {
		return nil, nil, ErrTokenRemoved
	}

	if v.paused {
		return nil, nil, ErrDepositVaultPaused
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
