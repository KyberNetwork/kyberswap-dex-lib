package hooklet

import (
	"context"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
	return false, u256.U0, false, u256.U0
}

func (h *baseHooklet) AfterSwap(_ *SwapParams) {}

func (h *baseHooklet) CloneState() IHooklet {
	return h
}
