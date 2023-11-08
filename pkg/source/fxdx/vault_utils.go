package fxdx

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

type VaultUtils struct {
	vault *Vault
}

func NewVaultUtils(vault *Vault) *VaultUtils {
	return &VaultUtils{
		vault: vault,
	}
}

func (u *VaultUtils) GetSwapFeeBasisPoints(tokenIn string, tokenOut string, usdfAmount *big.Int) *big.Int {
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

	feeBasisPoints0 := u.GetFeeBasisPoints(tokenIn, usdfAmount, baseBps, taxBps, true)
	feeBasisPoints1 := u.GetFeeBasisPoints(tokenOut, usdfAmount, baseBps, taxBps, false)

	if feeBasisPoints0.Cmp(feeBasisPoints1) > 0 {
		return feeBasisPoints0
	} else {
		return feeBasisPoints1
	}
}

func (u *VaultUtils) GetFeeBasisPoints(token string, usdfDelta *big.Int, feeBasisPoints *big.Int, taxBasisPoints *big.Int, increment bool) *big.Int {
	if !u.vault.HasDynamicFees {
		return feeBasisPoints
	}

	initialAmount := u.vault.USDFAmounts[token]
	nextAmount := new(big.Int).Add(initialAmount, usdfDelta)

	if !increment {
		if usdfDelta.Cmp(initialAmount) > 0 {
			nextAmount = integer.Zero()
		} else {
			nextAmount = new(big.Int).Sub(initialAmount, usdfDelta)
		}
	}

	targetAmount := u.vault.GetTargetUSDFAmount(token)

	if targetAmount.Cmp(integer.Zero()) == 0 {
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
			return integer.Zero()
		} else {
			return new(big.Int).Sub(feeBasisPoints, rebateBps)
		}
	}

	averageDiff := new(big.Int).Div(new(big.Int).Add(initialDiff, nextDiff), integer.Two())

	if averageDiff.Cmp(targetAmount) > 0 {
		averageDiff = targetAmount
	}

	taxBps := new(big.Int).Div(new(big.Int).Mul(taxBasisPoints, averageDiff), targetAmount)

	return new(big.Int).Add(feeBasisPoints, taxBps)
}
