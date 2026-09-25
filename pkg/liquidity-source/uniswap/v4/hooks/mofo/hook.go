// Package mofo implements MofoHook, the single Uniswap v4 hook every coin launched on mofo (https://mofo.zone)
// trades through on Robinhood chain. Verified on Blockscout and RobinScan at
// 0x665C52D02Ddc506dfb3158C100Edc77FE412a8c0. Enabled callbacks: beforeInitialize, beforeAddLiquidity,
// beforeSwap, afterSwap -- no return-delta flags, no hookData, no owner, not upgradeable.
//
//   - beforeSwap returns the pool's LP fee OR'd with OVERRIDE_FEE_FLAG. Every pool key carries the dynamic-fee
//     flag and slot0.lpFee stays 0, so the fee exists only as this override: the pool's feePips (frozen at
//     initialisation, at most 20%), except during the first 3 seconds after the pool was created, when it steps
//     down from 99% (see currentFeePips).
//   - afterSwap reverts BelowBirthPrice if a swap would leave the coin priced below the price the pool was
//     initialised at. No hook fee is taken. The pool's only liquidity range has one edge exactly at that birth
//     price, so the birth tick is the outermost initialised tick on the sell side. The simulator therefore refuses
//     a sell larger than the range can absorb (the shared tick math errors past the last tick) rather than quote a
//     swap the hook would revert; the one gap is an exact-output sell within a few raw units of the pool's whole
//     capacity, from the shared math's rounding. The price limit it hands the executor (GetSqrtPriceLimit, one
//     unit inside that tick) stops a swap before birth. A swap sent without a price limit, after the price has
//     moved against the quote, reverts BelowBirthPrice instead; nothing is lost.
package mofo

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
	ErrPoolNotRegistered = errors.New("mofo: pool not registered with the hook")
	ErrInvalidFee        = errors.New("mofo: pool fee outside (0, 20%]")
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
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Mofo}}
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

	if raw.FeePips == nil || raw.FeePips.Sign() == 0 {
		// not initialised through the factory (cannot happen for a pool that exists with this hook): keep nothing,
		// so BeforeSwap refuses to price it rather than quote it at slot0's fee of 0
		return json.Marshal(Extra{})
	}
	h.Extra = Extra{FeePips: uint32(raw.FeePips.Uint64()), LaunchedAt: raw.LaunchedAt.Int64()}
	return json.Marshal(h.Extra)
}

// BeforeSwap mirrors MofoHook.beforeSwap: no deltas, only the LP fee override.
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
	}, nil
}

// currentFeePips mirrors MofoHook.currentFeePips for any swapper a router can be: 99% in the second the pool was
// created, stepping down linearly (floored, as in Solidity) to the pool's fee at 3 seconds. The hook's one exemption
// (the launch transaction's own opening buy, elapsed == 0) never applies to a routed swap. A clock behind the pool's
// launch time is treated as the launch second, the most expensive case.
func currentFeePips(feePips uint32, launchedAt, now int64) uint32 {
	elapsed := now - launchedAt
	if elapsed >= sniperSeconds {
		return feePips
	} else if elapsed < 0 {
		elapsed = 0
	}
	return uint32(sniperStartPips - (uint64(sniperStartPips-feePips)*uint64(elapsed))/sniperSeconds)
}
