// Package tickermemev4 registers Ticker's proxy with the existing Pons fee handler.
package tickermemev4

import (
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
)

type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	ponsv2.Extra
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddress)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: DexType}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return h.Extra.AfterSwap(params)
}
