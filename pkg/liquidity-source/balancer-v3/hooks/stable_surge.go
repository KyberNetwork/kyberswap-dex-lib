package hooks

import (
	"slices"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
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

	if params.Kind == shared.EXACT_IN {
		newBalances[params.IndexIn] = new(uint256.Int).Add(newBalances[params.IndexIn], params.AmountGivenScaled18)
		newBalances[params.IndexOut] = amtCalculatedScaled18.Sub(newBalances[params.IndexOut], amtCalculatedScaled18)
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

	newTotalImbalance, err := math.StableSurgeMedian.CalculateImbalance(balances)
	if err != nil || !h._isSurging(params.BalancesScaled18, newTotalImbalance) {
		return params.StaticSwapFeePercentage
	}

	tmp, err := math.FixPoint.DivDown(newTotalImbalance.Sub(newTotalImbalance, h.ThresholdPercentage),
		math.FixPoint.Complement(h.ThresholdPercentage))
	if err != nil {
		return params.StaticSwapFeePercentage
	}
	tmp, err = math.FixPoint.MulDown(newTotalImbalance.Sub(h.MaxSurgeFeePercentage, params.StaticSwapFeePercentage),
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
	oldTotalImbalance, err := math.StableSurgeMedian.CalculateImbalance(currentBalances)
	return err == nil && newTotalImbalance.Gt(oldTotalImbalance)
}
