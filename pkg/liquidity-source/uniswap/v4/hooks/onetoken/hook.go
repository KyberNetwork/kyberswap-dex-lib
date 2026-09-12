// Package onetoken implements the Uniswap v4 hook shared by every OneToken launch
// (onetokenhub.xyz; the deployed contract is verified as RampHook on each chain's
// explorer). A OneToken pool is a one-sided bonding curve whose fee is a plain
// dynamic LP fee: the hook's beforeSwap returns feePips | OVERRIDE_FEE_FLAG, so the
// PoolManager applies it as the pool's LP fee for that swap. Unlike b20/LaunchHook,
// there is no delta and no afterSwap: the fee overrides the LP fee symmetrically, so
// this handler only sets BeforeSwapResult.SwapFee (pool_simulator.go then swaps that
// fee in via cloned.V3Pool.Fee). The fee is the normal platform fee, an elevated flat
// snipe tax during the first snipeWindow seconds after launch, or a per-pool override,
// always hard-capped at MaxFeeBps.
package onetoken

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var ErrNotLaunched = errors.New("onetoken: pool has no launchTime yet (not launched), refusing to quote at an unknown fee")

// Extra is the pricing-relevant state read from the OneToken hook. FeeBps/SnipeTaxBps/
// SnipeWindow are the hook-global params (owner-settable, rarely change); LaunchTime
// and OverrideBps are per-pool. Only the fee matters to a swap: creator/admin fee
// splits happen after collection and never affect the swap-visible amount.
type Extra struct {
	FeeBps      int64 `json:"f"`
	SnipeTaxBps int64 `json:"s"`
	SnipeWindow int64 `json:"w"`
	LaunchTime  int64 `json:"lt,omitempty"`
	OverrideBps int64 `json:"o,omitempty"` // 0 = no per-pool override
}

// NowFn is a var so tests can pin the snipe-window clock.
var NowFn = func() int64 { return time.Now().Unix() }

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4OneToken}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

// Track reads the pool's launchTime and fee override plus the hook-global fee params
// in one multicall. launchTime is set on the pool's first liquidity add and never
// changes; the globals are re-read every pass (cheap) so a later owner setFeeBps /
// setSnipeTax stays reflected. A zero launchTime means the pool has not launched
// (nothing to price), so we refuse rather than quote it fee-free.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	var (
		launchTime  = new(big.Int)
		overrideRaw uint16
		feeBps      uint16
		snipeTax    uint16
		snipeWindow = new(big.Int)
	)
	poolID := common.HexToHash(param.Pool.Address)
	target := hexutil.Encode(param.HookAddress[:])
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).SetOverrides(param.Overrides)
	req.AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "launchTime", Params: []any{poolID}}, []any{&launchTime})
	req.AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "poolFeeBpsOverride", Params: []any{poolID}}, []any{&overrideRaw})
	req.AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "feeBps", Params: nil}, []any{&feeBps})
	req.AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "snipeTaxBps", Params: nil}, []any{&snipeTax})
	req.AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "snipeWindow", Params: nil}, []any{&snipeWindow})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	if launchTime.Sign() == 0 {
		return nil, ErrNotLaunched
	}

	// poolFeeBpsOverride stores bps+1 so 0 = unset; decode back to a real bps value.
	var overrideBps int64
	if overrideRaw > 0 {
		overrideBps = int64(overrideRaw) - 1
	}
	h.Extra = Extra{
		FeeBps:      int64(feeBps),
		SnipeTaxBps: int64(snipeTax),
		SnipeWindow: snipeWindow.Int64(),
		LaunchTime:  launchTime.Int64(),
		OverrideBps: overrideBps,
	}
	return json.Marshal(h)
}

// EffectiveFeeBps mirrors the hook's effectiveFeeBps(): snipe window takes priority,
// then a per-pool override, then the global fee. Result is hard-capped at MaxFeeBps.
func (e *Extra) EffectiveFeeBps() int64 {
	fee := e.FeeBps
	if e.LaunchTime != 0 && NowFn() < e.LaunchTime+e.SnipeWindow {
		fee = e.SnipeTaxBps
	} else if e.OverrideBps > 0 {
		fee = e.OverrideBps
	}
	if fee > MaxFeeBps {
		fee = MaxFeeBps
	}
	return fee
}

// BeforeSwap overrides the LP fee with the current effective fee (the hook returns
// feePips | OVERRIDE_FEE_FLAG). V4 fee unit is pips (1e-6) and our bps is 1e-4, so
// pips = bps * 100. No delta, no direction check: the fee applies symmetrically to
// exact-in and exact-out, so this returns the same SwapFee for both.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.LaunchTime == 0 {
		return nil, ErrNotLaunched
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          uniswapv4.FeeAmount(h.EffectiveFeeBps() * feePipsPerBps),
	}, nil
}
