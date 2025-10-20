package hooks

import (
	"math"
	"math/big"
	"slices"

	"github.com/holiman/uint256"

	bmath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
)

type StableSurgeHook struct {
	NoOpHook

	MaxSurgeFeePercentage *uint256.Int
	ThresholdPercentage   *uint256.Int
}

func NewStableSurgeHook(maxSurgeFeePercentage, thresholdPercentage *uint256.Int) *StableSurgeHook {
	return &StableSurgeHook{
		MaxSurgeFeePercentage: maxSurgeFeePercentage,
		ThresholdPercentage:   thresholdPercentage,
	}
}

func (h *StableSurgeHook) OnComputeDynamicSwapFeePercentage(params shared.PoolSwapParams) (bool, *uint256.Int, error) {
	return true, h.getSurgeFeePercentage(params), nil
}

func (h *StableSurgeHook) getSurgeFeePercentage(params shared.PoolSwapParams) *uint256.Int {
	amtCalculatedScaled18, err := params.OnSwap(params)
	if err != nil {
		return params.StaticSwapFeePercentage
	}

	newBalances := slices.Clone(params.BalancesScaled18)

	if params.Kind == shared.ExactIn {
		// inflate imbalance to account for on-chain slippage
		balOutF := newBalances[params.IndexOut].Float64()
		amtOutF := balOutF * math.Log(3.2+32*amtCalculatedScaled18.Float64()/balOutF) / math.Log(3.2+32)
		amtOutB, _ := big.NewFloat(amtOutF).Int(nil)
		var inflatedAmt uint256.Int
		inflatedAmt.SetFromBig(amtOutB)
		newBalances[params.IndexOut] = new(uint256.Int).Sub(newBalances[params.IndexOut], &inflatedAmt)
		inflatedAmt.MulDivOverflow(params.AmountGivenScaled18, &inflatedAmt, amtCalculatedScaled18)
		newBalances[params.IndexIn] = inflatedAmt.Add(newBalances[params.IndexIn], &inflatedAmt)
	} else {
		newBalances[params.IndexIn] = amtCalculatedScaled18.Add(newBalances[params.IndexIn], amtCalculatedScaled18)
		newBalances[params.IndexOut] = new(uint256.Int).Sub(newBalances[params.IndexOut], params.AmountGivenScaled18)
	}

	return h._getSurgeFeePercentage(params, newBalances)
}

func (h *StableSurgeHook) _getSurgeFeePercentage(params shared.PoolSwapParams, balances []*uint256.Int) *uint256.Int {
	if h.MaxSurgeFeePercentage.Lt(params.StaticSwapFeePercentage) {
		return params.StaticSwapFeePercentage
	}

	newTotalImbalance, err := bmath.StableSurgeMedian.CalculateImbalance(balances)
	if err != nil || !h._isSurging(params.BalancesScaled18, newTotalImbalance) {
		return params.StaticSwapFeePercentage
	}

	tmp, err := bmath.FixPoint.DivDown(newTotalImbalance.Sub(newTotalImbalance, h.ThresholdPercentage),
		bmath.FixPoint.Complement(h.ThresholdPercentage))
	if err != nil {
		return params.StaticSwapFeePercentage
	}
	tmp, err = bmath.FixPoint.MulDown(newTotalImbalance.Sub(h.MaxSurgeFeePercentage, params.StaticSwapFeePercentage),
		tmp)
	if err != nil {
		return params.StaticSwapFeePercentage
	}
	return tmp.Add(params.StaticSwapFeePercentage, tmp)
}

func (h *StableSurgeHook) _isSurging(currentBalances []*uint256.Int, newTotalImbalance *uint256.Int) bool {
	if newTotalImbalance.IsZero() || !newTotalImbalance.Gt(h.ThresholdPercentage) {
		return false
	}
	oldTotalImbalance, err := bmath.StableSurgeMedian.CalculateImbalance(currentBalances)
	return err == nil && newTotalImbalance.Gt(oldTotalImbalance)
}
