package sellonly

import (
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook is the SellOnly / SellOnlyOne hook pair (both verified on Etherscan, canonical v4 PoolManager). They have
// beforeInitialize, beforeAddLiquidity, beforeDonate and beforeSwap permissions, take no fee and return no delta.
// A swap selling the hook's currency prices exactly like a hookless pool; every other swap reverts unless the pool
// owner sends it, so routers only have to refuse the other direction.
type Hook struct {
	*uniswapv4.BaseHook
	// sellsZero is true when the permitted swap is zeroForOne (SellOnly), false when oneForZero (SellOnlyOne).
	sellsZero bool
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		BaseHook:  &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4SellOnly},
		sellsZero: param.HookAddress == SellOnly,
	}
}, SellOnly, SellOnlyOne)

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if params.ZeroForOne != h.sellsZero {
		return nil, ErrWrongDirection
	}
	return h.BaseHook.BeforeSwap(params)
}

func (h *Hook) CloneState() uniswapv4.Hook {
	return h
}
