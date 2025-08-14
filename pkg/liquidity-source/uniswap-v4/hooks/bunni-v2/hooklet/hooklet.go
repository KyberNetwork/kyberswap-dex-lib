package hooklet

import (
	"context"

	"github.com/holiman/uint256"
)

type IHooklet interface {
	Track(context.Context, HookletParams) (string, error)
	BeforeSwap(*SwapParams) (feeOverriden bool, fee *uint256.Int, priceOverridden bool, sqrtPriceX96 *uint256.Int)
	AfterSwap(*SwapParams)
	CloneState() IHooklet
}

type baseHooklet struct{}

func NewBaseHooklet(_ string) *baseHooklet {
	return &baseHooklet{}
}

func (h *baseHooklet) Track(_ context.Context, _ HookletParams) (string, error) {
	return "", nil
}

func (h *baseHooklet) BeforeSwap(_ *SwapParams) (bool, *uint256.Int, bool, *uint256.Int) {
	return false, new(uint256.Int), false, new(uint256.Int)
}

func (h *baseHooklet) AfterSwap(_ *SwapParams) {}

func (h *baseHooklet) CloneState() IHooklet {
	return h
}
