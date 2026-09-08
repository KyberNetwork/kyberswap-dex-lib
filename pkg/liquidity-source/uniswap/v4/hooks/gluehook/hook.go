package gluehook

import (
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook of GlueHook (https://gluehook.trade) — a free, keyless, general-purpose hook that gives a
// pool a permissionless buyback pot (it buys after buys and can absorb sells) plus a
// self-compounding LP position owned by the hook itself.
//
// Quoting is a pure passthrough: the hook NEVER changes the swapper's amounts.
//   - Pump executes after the swap, spending the pot's own balance;
//   - Shield absorbs a sell while paying the seller the pool's EXACT output
//     (fee and tick impact included), so the quoted output is identical either way;
//   - auto-harvest/compound only moves the hook's own LP fees.
//
// Every hook action is wrapped in try/catch — a swap can never revert on hook state or
// settings. The only difference from a vanilla pool is gas, budgeted at the audited
// worst case here.
type Hook struct {
	*uniswapv4.BaseHook
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4GlueHook},
	}
}, HookAddresses...)

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	result, err := h.BaseHook.BeforeSwap(params)
	if err != nil {
		return nil, err
	}
	result.Gas = maxHookGas
	return result, nil
}
