package fxdx

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

type FeeUtilsV2 struct {
	Address string `json:"address"`

	IsInitialized           bool     `json:"isInitialized"`
	IsActive                bool     `json:"isActive"`
	FeeMultiplierIfInactive *big.Int `json:"feeMultiplierIfInactive"`
	HasDynamicFees          bool     `json:"hasDynamicFees"`

	TaxBasisPoints     map[string]*big.Int `json:"taxBasisPoints"`
	SwapFeeBasisPoints map[string]*big.Int `json:"swapFeeBasisPoints"`

	Vault *Vault `json:"-"`
}

func NewFeeUtilsV2() *FeeUtilsV2 {
	return &FeeUtilsV2{
		TaxBasisPoints:     make(map[string]*big.Int),
		SwapFeeBasisPoints: make(map[string]*big.Int),
	}
}

const (
	feeUtilsV2MethodGetStates = "getStates"
	feeUtilsV2IsInitialized   = "isInitialized"
)

func (f *FeeUtilsV2) GetSwapFeeBasisPoints(
	tokenIn string,
	tokenOut string,
	usdfAmount *big.Int,
) (*big.Int, error) {
	if !f.IsInitialized {
		return nil, ErrFeeUtilsV2IsNotInitialized
	}

	feesBasisPoints0 := f.getFeeBasisPoints(tokenIn, usdfAmount, f.SwapFeeBasisPoints[tokenIn], f.TaxBasisPoints[tokenIn], true)
	feesBasisPoints1 := f.getFeeBasisPoints(tokenOut, usdfAmount, f.SwapFeeBasisPoints[tokenOut], f.TaxBasisPoints[tokenOut], false)

	if feesBasisPoints0.Cmp(feesBasisPoints1) > 0 {
		return feesBasisPoints0, nil
	}

	return feesBasisPoints1, nil
}

func (f *FeeUtilsV2) getFeeBasisPoints(
	token string,
	usdfDelta *big.Int,
	feeBasisPoints *big.Int,
	taxBasisPoints *big.Int,
	increment bool,
) *big.Int {
	feeBps := feeBasisPoints
	if !f.IsActive {
		feeBps = new(big.Int).Mul(feeBasisPoints, f.FeeMultiplierIfInactive)
	}

	if !f.HasDynamicFees {
		return feeBps
	}

	initialAmount := f.Vault.USDFAmounts[token]
	nextAmount := new(big.Int).Add(initialAmount, usdfDelta)
	if !increment {
		if usdfDelta.Cmp(initialAmount) > 0 {
			nextAmount = integer.Zero()
		} else {
			nextAmount = new(big.Int).Sub(initialAmount, usdfDelta)
		}
	}

	targetAmount := f.Vault.GetTargetUSDFAmount(token)
	if targetAmount.Cmp(integer.Zero()) == 0 {
		return feeBps
	}

	initialDiff := new(big.Int).Abs(new(big.Int).Sub(initialAmount, targetAmount))
	nextDiff := new(big.Int).Abs(new(big.Int).Sub(nextAmount, targetAmount))

	if nextDiff.Cmp(initialDiff) < 0 {
		rebateBps := new(big.Int).Div(new(big.Int).Mul(taxBasisPoints, initialDiff), targetAmount)
		if rebateBps.Cmp(feeBps) > 0 {
			return integer.Zero()
		}
		return new(big.Int).Sub(feeBps, rebateBps)
	}

	averageDiff := new(big.Int).Div(new(big.Int).Add(initialDiff, nextDiff), integer.Two())
	if averageDiff.Cmp(targetAmount) > 0 {
		averageDiff = targetAmount
	}
	taxBps := new(big.Int).Div(new(big.Int).Mul(taxBasisPoints, averageDiff), targetAmount)
	return new(big.Int).Add(feeBps, taxBps)
}
