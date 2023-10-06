package gmxglp

import (
	"math/big"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// VaultUtils
// https://github.com/gmx-io/gmx-contracts/blob/master/contracts/core/VaultUtils.sol
type VaultUtils struct {
	vault *Vault
}

func NewVaultUtils(vault *Vault) *VaultUtils {
	return &VaultUtils{
		vault: vault,
	}
}

func (u *VaultUtils) GetSwapFeeBasisPoints(tokenIn string, tokenOut string, usdgAmount *big.Int) *big.Int {
	isStableSwap := u.vault.StableTokens[tokenIn] && u.vault.StableTokens[tokenOut]

	var baseBps *big.Int
	if isStableSwap {
		baseBps = u.vault.StableSwapFeeBasisPoints
	} else {
		baseBps = u.vault.SwapFeeBasisPoints
	}

	var taxBps *big.Int
	if isStableSwap {
		taxBps = u.vault.StableTaxBasisPoints
	} else {
		taxBps = u.vault.TaxBasisPoints
	}

	feeBasisPoints0 := u.GetFeeBasisPoints(tokenIn, usdgAmount, baseBps, taxBps, true)
	feeBasisPoints1 := u.GetFeeBasisPoints(tokenOut, usdgAmount, baseBps, taxBps, false)

	if feeBasisPoints0.Cmp(feeBasisPoints1) > 0 {
		return feeBasisPoints0
	} else {
		return feeBasisPoints1
	}
}

func (u *VaultUtils) GetBuyUsdgFeeBasisPoints(tokenIn string, usdgAmount *big.Int) *big.Int {
	return u.GetFeeBasisPoints(tokenIn, usdgAmount, u.vault.MintBurnFeeBasicPoints, u.vault.TaxBasisPoints, true)
}

func (u *VaultUtils) GetSellUsdgFeeBasisPoints(token string, usdgAmount *big.Int) *big.Int {
	return u.GetFeeBasisPoints(token, usdgAmount, u.vault.MintBurnFeeBasicPoints, u.vault.TaxBasisPoints, false)
}

func (u *VaultUtils) GetFeeBasisPoints(token string, usdgDelta *big.Int, feeBasisPoints *big.Int, taxBasisPoints *big.Int, increment bool) *big.Int {
	if !u.vault.HasDynamicFees {
		return feeBasisPoints
	}

	initialAmount := u.vault.USDGAmounts[token]
	nextAmount := new(big.Int).Add(initialAmount, usdgDelta)

	if !increment {
		if usdgDelta.Cmp(initialAmount) > 0 {
			nextAmount = constant.ZeroBI
		} else {
			nextAmount = new(big.Int).Sub(initialAmount, usdgDelta)
		}
	}

	targetAmount := u.vault.GetTargetUSDGAmount(token)

	if targetAmount.Cmp(constant.ZeroBI) == 0 {
		return feeBasisPoints
	}

	var initialDiff *big.Int
	if initialAmount.Cmp(targetAmount) > 0 {
		initialDiff = new(big.Int).Sub(initialAmount, targetAmount)
	} else {
		initialDiff = new(big.Int).Sub(targetAmount, initialAmount)
	}

	var nextDiff *big.Int
	if nextAmount.Cmp(targetAmount) > 0 {
		nextDiff = new(big.Int).Sub(nextAmount, targetAmount)
	} else {
		nextDiff = new(big.Int).Sub(targetAmount, nextAmount)
	}

	if nextDiff.Cmp(initialDiff) < 0 {
		rebateBps := new(big.Int).Div(new(big.Int).Mul(taxBasisPoints, initialDiff), targetAmount)

		if rebateBps.Cmp(feeBasisPoints) > 0 {
			return constant.ZeroBI
		} else {
			return new(big.Int).Sub(feeBasisPoints, rebateBps)
		}
	}

	averageDiff := new(big.Int).Div(new(big.Int).Add(initialDiff, nextDiff), constant.Two)

	if averageDiff.Cmp(targetAmount) > 0 {
		averageDiff = targetAmount
	}

	taxBps := new(big.Int).Div(new(big.Int).Mul(taxBasisPoints, averageDiff), targetAmount)

	return new(big.Int).Add(feeBasisPoints, taxBps)
}
