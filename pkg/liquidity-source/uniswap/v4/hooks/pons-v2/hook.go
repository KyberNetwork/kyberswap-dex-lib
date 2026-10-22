// Package ponsv2 implements PonsV2MemeHook, the singleton Uniswap v4 hook shared by
// every graduated pons-v2 pool on Robinhood chain (see the pons-v2 package for the
// pre-graduation bonding curve). Verified via Sourcify at
// 0xE5e702641Ea86F4ae6cC3cDaeD2B886f976Be044: only afterSwap (+ afterSwapReturnDelta)
// is enabled -- the hook takes hookFeeBps + creatorTaxBps of the swap's unspecified
// currency straight out of the pool manager's flash-accounting ledger via a single
// `take`, so the fee lands on the realized output of an exact-in swap (on the input
// of an exact-out one, which Kyber never executes). Both bps are frozen per pool at
// registerPool time, read here via the `launches` public mapping getter.
//
// Track, BeforeSwap and AfterSwap live on Extra so another deployment of the same PonsV2MemeHook
// bytecode (hooks/coocoo) can embed it under its own exchange and address list, the
// way hooks/o1 reuses hooks/b20.
package ponsv2

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Extra struct {
	Registered bool  `json:"r,omitempty"`
	FeeBps     int64 `json:"f,omitempty"`
}

// Hook embeds BaseHook by value, not the Hook interface, so a simulator survives the msgpack round trip.
type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Exchange: valueobject.ExchangeUniswapV4PonsV2}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// Track reads the pool's frozen hookFeeBps + creatorTaxBps from the hook's `launches`
// getter (see Extra.Track).
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if err := h.Extra.Track(ctx, param); err != nil {
		return nil, err
	}
	return json.Marshal(h)
}

// Track fills e from the hook's `launches` getter for param.Pool. Both bps are snapshotted
// immutably at registerPool time (later owner-level fee updates only affect pools registered
// afterward), so an already-registered pool is never re-fetched.
func (e *Extra) Track(ctx context.Context, param *uniswapv4.HookParam) error {
	if e.Registered {
		return nil
	}

	var raw launchRaw
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "launches",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&raw}).Call(); err != nil {
		return err
	}

	*e = Extra{
		Registered: raw.Registered,
		FeeBps:     int64(raw.HookFeeBps) + int64(raw.CreatorTaxBps),
	}
	return nil
}

// Delegate rather than rely on promotion: Hook embeds both BaseHook
// and Extra at the same depth, both with BeforeSwap/AfterSwap, so an unqualified call
// would otherwise be an ambiguous selector.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return h.Extra.AfterSwap(params)
}

// CanBeforeSwap is true although the on-chain hook has no beforeSwap: only the simulator
// calls it, to gross the output up in CalcAmountIn (see Extra.BeforeSwap).
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return h.Extra.BeforeSwap(params)
}

// BeforeSwap serves CalcAmountIn only, which reverses an exact-in swap. The fee comes out of
// the realized output, so the pool must output the requested amount grossed up by the fee.
func (e *Extra) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	delta := bignumber.ZeroBI
	if !params.CalcOut && e.Registered && e.FeeBps > 0 {
		if e.FeeBps >= basisPoints {
			return nil, uniswapv4.ErrInvalidFee
		}
		gross := uniswapv4.GrossBeforeFee(params.AmountSpecified, big.NewInt(e.FeeBps), big.NewInt(basisPoints))
		delta = gross.Sub(gross, params.AmountSpecified)
	}
	return &uniswapv4.BeforeSwapResult{DeltaSpecified: delta, DeltaUnspecified: bignumber.ZeroBI}, nil
}

// AfterSwap mirrors PonsV2MemeHook._afterSwap for exact-in: the fee is a flat feeBps of the
// realized output, taken directly from the pool manager. CalcAmountIn reverses an exact-in
// swap (never the contract's exact-out branch, which charges the input), and BeforeSwap
// already priced the output fee, so it charges nothing there.
func (e *Extra) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if !params.CalcOut || !e.Registered || e.FeeBps == 0 {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.MulDivDown(new(big.Int), params.AmountOut, big.NewInt(e.FeeBps), big.NewInt(basisPoints)),
	}, nil
}
