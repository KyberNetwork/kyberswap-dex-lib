package hooks

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type directionalFeeHook struct {
	NoOpHook

	staticSwapFeePercentage *uint256.Int
}

func NewDirectionalFeeHook(staticSwapFeePercentage *uint256.Int) *directionalFeeHook {
	return &directionalFeeHook{
		staticSwapFeePercentage: staticSwapFeePercentage,
	}
}

func (h *directionalFeeHook) OnComputeDynamicSwapFeePercentage(param shared.PoolSwapParams) (bool, *uint256.Int, error) {
	calculatedSwapFeePercentage, err := h.calculatedExpectedSwapFeePercentage(param.BalancesLiveScaled18[param.IndexIn],
		param.BalancesLiveScaled18[param.IndexOut], param.AmountGivenScaled18)
	if err != nil {
		return false, nil, err
	}

	if calculatedSwapFeePercentage.Gt(h.staticSwapFeePercentage) {
		return true, calculatedSwapFeePercentage, nil
	}

	return false, new(uint256.Int).Set(h.staticSwapFeePercentage), nil
}

func (h *directionalFeeHook) calculatedExpectedSwapFeePercentage(balanceIn, balanceOut, swapAmount *uint256.Int) (*uint256.Int, error) {
	finalBalanceTokenIn, err := math.FixPoint.Add(balanceIn, swapAmount)
	if err != nil {
		return nil, err
	}

	finalBalanceTokenOut, err := math.FixPoint.Sub(balanceOut, swapAmount)
	if err != nil {
		return nil, err
	}

	if finalBalanceTokenIn.Gt(finalBalanceTokenOut) {
		diff, err := math.FixPoint.Sub(finalBalanceTokenIn, finalBalanceTokenOut)
		if err != nil {
			return nil, err
		}

		totalLiquidity, err := math.FixPoint.Add(finalBalanceTokenIn, finalBalanceTokenOut)
		if err != nil {
			return nil, err
		}

		diff, err = math.FixPoint.DivDown(diff, totalLiquidity)
		return diff, err
	}

	return math.ZERO, nil
}
