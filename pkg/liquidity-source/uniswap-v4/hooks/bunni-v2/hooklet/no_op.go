package hooklet

import (
	"context"

	"github.com/holiman/uint256"
)

type noopHooklet struct{}

func NewNoopHooklet(_ string) *noopHooklet {
	return &noopHooklet{}
}

func (h *noopHooklet) Track(_ context.Context, _ HookletParams) (string, error) {
	return "", nil
}

func (h *noopHooklet) BeforeSwap(_ *SwapParams) (bool, *uint256.Int, bool, *uint256.Int) {
	return false, new(uint256.Int), false, new(uint256.Int)
}

func (h *noopHooklet) AfterSwap(_ *SwapParams) {}
