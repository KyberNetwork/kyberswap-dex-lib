// Package reafund registers the Uniswap v4 hook of REA.fund (https://rea.fund), a launchpad on
// Robinhood Chain for tokens of open-source software. REA.fund deploys the Pons V2 contracts
// unmodified and launches through their factory, so a launch trades on a pons-v2 bonding curve
// until it graduates into a v4 pool behind REA.fund's own PonsV2MemeHook. That hook is Pons'
// hook (hooks/pons-v2) with another fee escrow in its immutables, and charges what Pons' does:
// afterSwap takes hookFeeBps + creatorTaxBps of the unspecified currency, both fixed for the
// pool when it registers. The pricing is ponsv2.Extra; this package gives the deployment its
// own exchange.
package reafund

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Reafund}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// Hook holds BaseHook by value so a pool simulator keeps it through the msgpack cache.
type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	ponsv2.Extra
}

// Track reads the pool's fees from the hook's `launches` getter until the pool is registered;
// after that they never change.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if err := h.Extra.Track(ctx, param); err != nil {
		return nil, err
	}
	return json.Marshal(h)
}

// CanBeforeSwap is true for the simulator only: the contract has no beforeSwap, but CalcAmountIn
// needs ponsv2.Extra.BeforeSwap to gross the requested output up by the fee.
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

// BaseHook and ponsv2.Extra both have BeforeSwap and AfterSwap; these pick Extra's.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return h.Extra.BeforeSwap(params)
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return h.Extra.AfterSwap(params)
}
