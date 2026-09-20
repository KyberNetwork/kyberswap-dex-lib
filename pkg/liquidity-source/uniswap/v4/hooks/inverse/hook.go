package inverse

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook simulates the complete exact-input trade, including the native v4 swap
// and both denomination corrections. v4's ordinary curve must not price it twice.
type Hook struct {
	uniswapv4.BaseHook
	State Extra
}

// Hash-verified fee-aware deployment on Robinhood (4663); see testdata/deployment.json.
// The historical live hook has different semantics and MUST NOT be added here.
var HookAddresses = []common.Address{common.HexToAddress("0x9bb08b8473d09eb235039fb8e3cc136d9787aaec")}
var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddresses...)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Inverse}}
	if param == nil {
		return h
	}
	if err := param.HookExtra.Unmarshal(&h.State); err != nil {
		h.State = Extra{}
	}
	return h
}
func (h *Hook) AllowEmptyTicks() bool             { return true }
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }
func (h *Hook) CanAfterSwap(common.Address) bool  { return false }
func (h *Hook) BeforeSwap(p *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if p == nil {
		return nil, ErrBounds
	}
	if !p.CalcOut {
		return nil, ErrExactOutput
	}
	out, next, err := quote(h.State, p.ZeroForOne, p.AmountSpecified)
	if err != nil {
		return nil, err
	}
	return &uniswapv4.BeforeSwapResult{DeltaSpecified: new(big.Int).Set(p.AmountSpecified), DeltaUnspecified: new(big.Int).Neg(out), SwapFee: 3000, Gas: 900_000, SwapInfo: SwapInfo{Before: h.State, After: next}}, nil
}
func (h *Hook) CloneState() uniswapv4.Hook { c := *h; return &c }
func (h *Hook) UpdateBalance(info any) {
	si, ok := info.(SwapInfo)
	if !ok || si.Before != h.State {
		return
	}
	h.State = si.After
}
func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if !h.State.Live {
		return entity.PoolReserves{"0", "0"}, nil
	}
	if err := h.State.validate(); err != nil {
		return nil, err
	}
	inv := md(h.State.ReserveShares.ToBig(), h.State.Index.ToBig(), ray).String()
	q := h.State.ReserveQuote.Dec()
	if h.State.Inverse0 {
		return entity.PoolReserves{inv, q}, nil
	}
	return entity.PoolReserves{q, inv}, nil
}

// PoolState replaces the base simulator's ordinary reserve deltas after a
// rebase and refreshes native state used for pool metadata.
func (h *Hook) PoolState() uniswapv4.HookPoolState {
	s := h.State
	inv := md(s.ReserveShares.ToBig(), s.Index.ToBig(), ray)
	r := [2]*big.Int{inv, s.ReserveQuote.ToBig()}
	if !s.Inverse0 {
		r[0], r[1] = r[1], r[0]
	}
	return uniswapv4.HookPoolState{Reserves: r, SqrtPriceX96: s.SqrtPriceX96, Liquidity: s.Liquidity, Tick: s.Tick}
}
