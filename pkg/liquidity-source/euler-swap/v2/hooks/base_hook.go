package hooks

import (
	"context"
	"errors"
)

var (
	ErrHookNotSupported = errors.New("hook not supported")
	ErrHookCallFailed   = errors.New("hook call failed")
)

type BaseHook struct{}

func NewBaseHook(param *HookParam) *BaseHook {
	return &BaseHook{}
}

func (h *BaseHook) GetFee(params *GetFeeParams) (uint64, error) {
	return 0, ErrHookCallFailed
}

func (h *BaseHook) BeforeSwap(_ *BeforeSwapParams) error {
	return nil
}

func (h *BaseHook) AfterSwap(_ *AfterSwapParams) error {
	return nil
}

func (h *BaseHook) Track(ctx context.Context, param *HookParam) (string, error) {
	return "", nil
}

func (h *BaseHook) CloneState() Hook {
	cloned := *h
	return &cloned
}
