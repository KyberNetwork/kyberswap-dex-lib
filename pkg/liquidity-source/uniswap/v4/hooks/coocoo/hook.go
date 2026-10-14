// Package coocoo registers CooCoo's deployments of PonsV2MemeHook, the Uniswap v4 hook
// behind every graduated launch of the CooCoo launchpad (https://www.coocoo.party) on
// Robinhood Chain and on Arc. CooCoo runs the Pons V2 contracts byte for byte: its
// factories emit the TokenLaunched event the pons-v2 source decodes, and each hook is
// the runtime code of Pons' own hook (hooks/pons-v2) apart from the immutable fee-escrow
// address, so the swap-visible fee is exactly pons-v2's -- afterSwap takes hookFeeBps +
// creatorTaxBps of the unspecified currency, both frozen per pool at registerPool time
// and read through the `launches` getter. This package only gives those deployments
// their own exchange and address list; the pricing is ponsv2.Extra's, reused the way
// hooks/o1 reuses hooks/b20.
package coocoo

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4CooCoo}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// Hook embeds BaseHook by value, not the Hook interface, so a simulator survives the msgpack round trip.
type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	ponsv2.Extra
}

// Track reads the pool's frozen hookFeeBps + creatorTaxBps from the hook's `launches`
// getter (ponsv2.Extra.Track); an already-registered pool is never re-fetched.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if err := h.Extra.Track(ctx, param); err != nil {
		return nil, err
	}
	return json.Marshal(h)
}

// Delegate rather than rely on promotion: Hook embeds both BaseHook
// and ponsv2.Extra at the same depth, both with BeforeSwap/AfterSwap, so an unqualified
// call would otherwise be an ambiguous selector.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return h.Extra.AfterSwap(params)
}

// CanBeforeSwap is true although the on-chain hook has no beforeSwap: only the simulator
// calls it, to gross the output up in CalcAmountIn (see ponsv2.Extra.BeforeSwap).
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return h.Extra.BeforeSwap(params)
}
