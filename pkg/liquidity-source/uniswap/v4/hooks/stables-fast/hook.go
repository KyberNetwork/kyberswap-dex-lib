// Package stablesfast implements ProtocolFeeHook, the singleton Uniswap v4 hook shared by every
// Stables asset market on Robinhood chain. Source: the Stables protocol contracts,
// src/markets/ProtocolFeeHook.sol.
//
// Permission bits 0x00CC: beforeSwap, afterSwap and both return-deltas. Only two of those four
// are exercised, and that is deliberate — see the note on beforeSwapReturnDelta below.
//
// The ENTIRE fee is taken in afterSwap, off the UNSPECIFIED leg of the swap, measured from the
// BalanceDelta the pool actually produced:
//
//   - exact-in  (CalcOut): the unspecified leg is the OUTPUT, so the trader receives
//     amountOut - floor(amountOut*feePips/1e6);
//   - exact-out (CalcIn):  the unspecified leg is the INPUT, so the trader pays
//     amountIn  + floor(amountIn*feePips/1e6).
//
// One rule covering both directions, so there is no swap type to flip into to escape the fee.
//
// `beforeSwap` returns a ZERO delta unconditionally and never charges anything. It exists only
// to write a V3-style oracle observation. The contract moved the exact-input skim out of it on
// 2026-09-19 for a reason that matters to an aggregator specifically: beforeSwap runs before
// pool.swap, so the only quantity available to it is the amount the caller ASKED for. A caller
// passing a binding sqrtPriceLimitX96 gets a partial fill, and the old ordering billed them on
// notional that never traded. Charging in afterSwap means a partial fill is charged on the
// fill. A router that sets its own price limit is precisely the caller that hit this, so do
// not reintroduce an input-side term here.
//
// Exact-output is NOT a gross-up. The hook charges its pips on the pool's realised (net) input,
// so the total is net + floor(net*fee/1e6), never net*fee/(1e6-fee). The asymmetry against the
// exact-input path is real and is worth roughly the square of the fee — about 25 pips at the
// shipped 0.50% rate. The protocol's own MarketLens._grossInputFor documents it from the other
// side, and carries two correction terms for exactly this reason.
//
// The oracle write runs for a registered pool in either direction and at any rate, so
// registration and the rate are tracked separately rather than folded into one number.
//
// The rate is read live rather than cached. It is per-pool owner-mutable up to MAX_FEE_PIPS,
// and `feePipsFor` additionally returns zero for every pool while the protocol guard is halted
// — a pause zeroes the skim rather than reverting, because a reverting hook would take the
// pools offline for every trader rather than just stopping the protocol's cut. A rate INCREASE
// is announced FEE_INCREASE_DELAY (one hour) ahead and applied by a separate permissionless
// commit, so a rate read here is good for at least an hour; a DECREASE applies immediately.
//
// The LP fee is a separate matter and this plugin never touches it. Markets created from
// 2026-09-21 on may carry Uniswap's DYNAMIC_FEE_FLAG (PoolKey.fee = 0x800000); for those the
// rate LPs earn is the stored slot0.lpFee, seeded at 5_000 by registration and moved by
// ProtocolFeeHook.setPoolLpFee — the owner, or one keeper address it authorises — inside
// 100..50_000 pips, in both directions, with no expiry. The keeper follows an off-chain
// calendar and reference-price model; nothing runs per swap, and beforeSwap returns no fee
// override, which is why BeforeSwapResult.SwapFee is left at zero here. The tracker's slot0
// read is therefore the source of truth for the LP fee on every cycle — PoolManager emits no
// event for updateDynamicLPFee, though the hook emits PoolLpFeeUpdated(poolId, feePips).
// Older static-tier pools are unaffected: the flag is part of pool identity and cannot be
// added to an existing pool.
package stablesfast

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Extra struct {
	// FeePips is ProtocolFeeHook.feePipsFor(poolId) as of the last Track. Zero is a real
	// rate: a pool the owner pinned at zero, or any pool while the guard is halted. Either
	// way the pool still trades.
	FeePips int64 `json:"f,omitempty"`

	// Registered is whether the factory ever registered this pool with the hook. An
	// unregistered pool keyed at this hook gets no skim AND no oracle write — the hook is a
	// no-op for it — which is a different thing from a registered pool at a zero rate.
	Registered bool `json:"r,omitempty"`

	Tracked bool `json:"t,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4StablesFast}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// Track re-reads both values every cycle: the rate moves with an owner setter and with the
// protocol guard's pause, neither of which emits an event on the pool.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	var (
		// `feePipsFor` returns uint24 and go-ethereum decodes anything but a native width
		// as *big.Int.
		feePips   *big.Int
		recipient common.Address
	)

	if param.RpcClient == nil {
		// No client on this refresh. Keeping the last reading beats dropping the pool out
		// of routing entirely; a pool that was never read stays untracked and simply does
		// not quote.
		if !h.Tracked {
			return nil, ErrPoolIsNotTracked
		}
		return json.Marshal(h)
	}

	target := hexutil.Encode(param.HookAddress[:])
	poolID := common.HexToHash(param.Pool.Address)
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: target, Method: "feePipsFor",
			Params: []any{poolID}}, []any{&feePips}).
		// A zero recipient is the hook's own "this pool is not ours" sentinel, and the same
		// condition that makes it skip the oracle.
		AddCall(&ethrpc.Call{ABI: hookABI, Target: target, Method: "feeRecipientOf",
			Params: []any{poolID}}, []any{&recipient}).
		Aggregate(); err != nil {
		return nil, err
	}

	if feePips == nil || feePips.Cmp(maxFeePips) > 0 {
		return nil, ErrFeeAboveMax
	}

	h.Extra = Extra{
		FeePips:    feePips.Int64(),
		Registered: recipient != (common.Address{}),
		Tracked:    true,
	}
	return json.Marshal(h)
}

// BeforeSwap mirrors ProtocolFeeHook.beforeSwap: the oracle write on every swap of a
// registered pool, and nothing else. Both deltas are zero in both directions — the contract
// returns BeforeSwapDeltaLibrary.ZERO_DELTA unconditionally.
func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.Tracked {
		return nil, ErrPoolIsNotTracked
	}

	result := &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}
	if h.Registered {
		result.Gas = gasObservation
	}
	return result, nil
}

// AfterSwap mirrors ProtocolFeeHook.afterSwap: the whole fee, both directions, charged on the
// unspecified leg of what the pool actually moved.
//
// AfterSwapResult.HookFee is defined by dex-lib as "CalcOut: out -= hook fee; CalcIn: in +=
// hook fee", which is the contract's rule stated in the simulator's own vocabulary: taking it
// off AmountOut for exact-in and off AmountIn for exact-out lands the charge on the unspecified
// currency in both cases.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if !h.Tracked {
		return nil, ErrPoolIsNotTracked
	}
	if !h.Registered || h.FeePips == 0 {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}

	return &uniswapv4.AfterSwapResult{
		HookFee: h.skim(lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn)),
		Gas:     gasAccrue,
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

// skim is FullMath.mulDiv on the contract, which truncates — matching it exactly is what keeps
// a quote from landing a unit above the fill.
func (h *Hook) skim(amount *big.Int) *big.Int {
	return bignumber.MulDivDown(new(big.Int), amount, big.NewInt(h.FeePips), pipsDenominator)
}
