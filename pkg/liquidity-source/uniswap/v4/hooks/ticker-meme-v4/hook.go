// Package tickermemev4 registers Ticker's proxy with the existing Pons fee handler.
package tickermemev4

import (
	"github.com/ethereum/go-ethereum/common"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
)

type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	ponsv2.Extra
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddress)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Exchange: DexType}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}

// CanBeforeSwap is true although the on-chain hook has no beforeSwap: only the simulator
// calls it, to gross the output up in CalcAmountIn (see ponsv2.Extra.BeforeSwap).
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return h.Extra.BeforeSwap(params)
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return h.Extra.AfterSwap(params)
}
