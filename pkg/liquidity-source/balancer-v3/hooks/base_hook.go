package hooks

import "github.com/holiman/uint256"

type baseHook struct{}

var BaseHook = &baseHook{}

func (h *baseHook) OnBeforeSwap() {}

func (h *baseHook) OnAfterSwap() {}

func (h *baseHook) OnComputeDynamicSwapFeePercentage(
	staticSwapFeePercentage,
	amountGivenScaled18,
	balanceIn,
	balanceOut *uint256.Int,
) (bool, *uint256.Int, error)
