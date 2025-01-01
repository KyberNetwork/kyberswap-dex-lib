package hooks

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/holiman/uint256"
)

type VeBALFeeDiscountHook struct {
	BaseHook
}

func NewVeBALFeeDiscountHook() *VeBALFeeDiscountHook {
	return &VeBALFeeDiscountHook{}
}

func (h *VeBALFeeDiscountHook) OnComputeDynamicSwapFeePercentage(
	staticSwapFeePercentage,
	amountGivenScaled18,
	balanceIn,
	balanceOut *uint256.Int,
) (bool, *uint256.Int, error) {
	calculatedSwapFeePercentage, err := h.calculatedExpectedSwapFeePercentage(balanceIn, balanceOut, amountGivenScaled18)
	if err != nil {
		return false, nil, err
	}

	if calculatedSwapFeePercentage.Gt(staticSwapFeePercentage) {
		return true, calculatedSwapFeePercentage, nil
	}

	return false, staticSwapFeePercentage, nil
}

func (h *VeBALFeeDiscountHook) calculatedExpectedSwapFeePercentage(balanceIn, balanceOut, swapAmount *uint256.Int) (*uint256.Int, error) {
	finalBalanceTokenIn, err := math.Add(balanceIn, swapAmount)
	if err != nil {
		return nil, err
	}

	finalBalanceTokenOut, err := math.Sub(balanceOut, swapAmount)
	if err != nil {
		return nil, err
	}

	if finalBalanceTokenIn.Gt(finalBalanceTokenOut) {
		diff, err := math.Sub(finalBalanceTokenIn, finalBalanceTokenOut)
		if err != nil {
			return nil, err
		}

		totalLiquidity, err := math.Add(finalBalanceTokenIn, finalBalanceTokenOut)
		if err != nil {
			return nil, err
		}

		diff, err = math.DivDown(diff, totalLiquidity)
		return diff, err
	}

	return math.ZERO, nil
}
