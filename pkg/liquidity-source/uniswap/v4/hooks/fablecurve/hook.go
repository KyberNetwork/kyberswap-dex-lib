// Package fablecurve implements FableHook (0x376aA4C2e48AC2498CaaAc06bd7A6b6ef373f8E0, Robinhood), the one Uniswap v4
// hook every coin launched on Fable (https://fablecurve.fun) trades through. Same pricing as MofoHook (package mofo).
//
//   - beforeSwap checkpoints the LiquidityLocker (gas only, gasBeforeSwap) and returns the pool's feePips (at most
//     20%, frozen at creation) as an override fee, stepping down from 99% over the first 3 s (currentFeePips).
//   - Launch pools: afterSwap reverts BelowBirthPrice if a swap prices the coin below the pool's birth price. The
//     locker's one position has an edge exactly at that price, so the tick math already errors past it and the
//     executor's price limit stays inside it. The hook doesn't enforce that layout; it relies on the Fable factory and
//     locker always adding the single launch position there. Mesh pools have no floor, full-range liquidity only.
//   - No hook fee and no deltas, so per-pool state is just the fee and the creation time.
package fablecurve

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrPoolNotRegistered = errors.New("fablecurve: pool not registered with the hook")
	ErrInvalidFee        = errors.New("fablecurve: pool fee outside (0, 20%]")
)

// NowFn is the clock the launch-window fee reads; a variable so tests can pin it.
var NowFn = func() int64 { return time.Now().Unix() }

// Extra is the hook state kept per pool. Both values are written once, in beforeInitialize, and never change.
type Extra struct {
	FeePips    uint32 `json:"f,omitempty"`
	LaunchedAt int64  `json:"l,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4FableCurve}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// Track reads the pool's feePips and launchedAt from the hook's `pools` getter. Both are immutable once the pool
// exists, so a pool that has been read once is never re-fetched.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.FeePips != 0 {
		return json.Marshal(h.Extra)
	}

	var raw poolInfoRaw
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "pools",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&raw}).Call(); err != nil {
		return nil, err
	}

	h.Extra = extraFromPoolInfo(&raw)
	return json.Marshal(h.Extra)
}

// extraFromPoolInfo keeps the fee and creation time of a pool the hook initialised. A pool id the hook never
// initialised reads as all zero (cannot happen for a pool that exists with this hook): it keeps nothing, so
// BeforeSwap refuses to price it rather than quote a fee the pool would not charge.
func extraFromPoolInfo(raw *poolInfoRaw) Extra {
	if raw.FeePips == nil || raw.FeePips.Sign() == 0 || !raw.FeePips.IsUint64() ||
		raw.LaunchedAt == nil || !raw.LaunchedAt.IsInt64() {
		return Extra{}
	}
	return Extra{FeePips: uint32(raw.FeePips.Uint64()), LaunchedAt: raw.LaunchedAt.Int64()}
}

// BeforeSwap mirrors FableHook.beforeSwap: no deltas, only the LP fee override, plus the locker checkpoint's gas.
func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.FeePips == 0 {
		return nil, ErrPoolNotRegistered
	} else if h.FeePips > maxFeePips {
		return nil, ErrInvalidFee
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          uniswapv4.FeeAmount(currentFeePips(h.FeePips, h.LaunchedAt, NowFn())),
		Gas:              gasBeforeSwap,
	}, nil
}

// currentFeePips mirrors FableHook.currentFeePips for any swapper a router can be: 99% in the second the pool was
// created, stepping down linearly (floored, as in Solidity) to the pool's fee at 3 seconds. The hook's one exemption
// (the launch transaction's own opening buy, elapsed == 0) never applies to a routed swap. A clock behind the pool's
// creation time is treated as the creation second, the most expensive case.
func currentFeePips(feePips uint32, launchedAt, now int64) uint32 {
	elapsed := now - launchedAt
	if elapsed >= sniperSeconds {
		return feePips
	} else if elapsed < 0 {
		elapsed = 0
	}
	return uint32(sniperStartPips - (uint64(sniperStartPips-feePips)*uint64(elapsed))/sniperSeconds)
}
