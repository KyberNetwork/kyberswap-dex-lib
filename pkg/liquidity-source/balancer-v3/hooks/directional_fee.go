package hooks

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type DirectionalFeeHook struct {
	NoOpHook
}

func NewDirectionalFeeHook() *DirectionalFeeHook {
	return &DirectionalFeeHook{}
}

func (h *DirectionalFeeHook) OnComputeDynamicSwapFeePercentage(params shared.PoolSwapParams) (bool, *uint256.Int,
	error) {
	calculatedSwapFeePercentage, err := h.calculatedExpectedSwapFeePercentage(params.BalancesScaled18[params.IndexIn],
		params.BalancesScaled18[params.IndexOut], params.AmountGivenScaled18)
	if err != nil {
		return false, nil, err
	}

	if calculatedSwapFeePercentage.Gt(params.StaticSwapFeePercentage) {
		return true, calculatedSwapFeePercentage, nil
	}

	return false, params.StaticSwapFeePercentage, nil
}

func (h *DirectionalFeeHook) calculatedExpectedSwapFeePercentage(balanceIn, balanceOut, swapAmount *uint256.Int) (*uint256.Int,
	error) {
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
