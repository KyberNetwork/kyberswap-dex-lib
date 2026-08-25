// Package b20 implements the Uniswap v4 hook shared by every B20 launchpad
// pool. See LaunchHook.sol on Base (0x985c14baa2A18316ffDA0AefB3a632faDFCA2acc):
// liquidity is permanently locked to the factory-seeded band (no third-party LP),
// and every swap pays a fee on the quote currency -- a base rate plus a
// linearly-decaying anti-snipe surcharge over the first antiSnipeWindowSeconds
// after launch. PoolKey.fee is always 0; all economics come from this overlay.
package b20

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

var (
	errNotRegistered     = errors.New("pool not registered with the B20 factory yet")
	errNotTracked        = errors.New("b20-launch: pool config not yet tracked, refusing to quote at an unknown fee")
	errCalcInUnsupported = errors.New("b20-launch: exact-out not supported (matches pool_simulator.go's CalcAmountOut-only scope)")
)

// Extra is the pricing-relevant subset of LaunchHook.sol's poolConfig -- frozen at
// registration except for the live block.timestamp comparison against LaunchTime.
// creator/platformTreasury/creatorBps/platformBps/referrerBps only affect how the
// FeeEscrow later splits the fee, not the swap-visible amount, so they're omitted.
type Extra struct {
	TokenIsCurrency0       bool  `json:"t0,omitempty"`
	BaseFeeBps             int64 `json:"bf"`
	AntiSnipeStartTotalBps int64 `json:"as,omitempty"`
	AntiSnipeWindowSeconds int64 `json:"aw,omitempty"`
	LaunchTime             int64 `json:"lt,omitempty"`
}

// nowFn is a var so tests can pin the anti-snipe decay clock.
var nowFn = func() int64 { return time.Now().Unix() }

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4B20}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

// Track reads the pool's frozen economics from the hook contract's poolConfig
// getter. It only needs to run once per pool -- poolConfig never changes after
// registration -- so an already-populated Extra with a non-zero LaunchTime is
// reused as-is instead of re-reading on every tracker pass.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.LaunchTime != 0 {
		return json.Marshal(h)
	}

	var raw poolConfigRaw
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    LaunchHookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "poolConfig",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&raw})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	if !raw.Initialized {
		return nil, errNotRegistered
	}

	h.Extra = Extra{
		TokenIsCurrency0:       raw.TokenIsCurrency0,
		BaseFeeBps:             int64(raw.BaseFeeBps),
		AntiSnipeStartTotalBps: int64(raw.AntiSnipeStartTotalBps),
		AntiSnipeWindowSeconds: int64(raw.AntiSnipeWindowSeconds),
		LaunchTime:             raw.LaunchTime.Int64(),
	}
	return json.Marshal(h)
}

// totalFeeBps mirrors LaunchHook.sol's _totalFeeBps: base + a linearly-decaying
// anti-snipe surcharge over the first AntiSnipeWindowSeconds after LaunchTime,
// capped at MaxTotalFeeBps. Evaluated against wall-clock time, same as every
// other time-decaying v4 hook in this package (see hooks/fairflow/hook.go).
func (h *Hook) totalFeeBps() int64 {
	elapsed := nowFn() - h.LaunchTime
	if elapsed >= h.AntiSnipeWindowSeconds {
		return h.BaseFeeBps
	}
	maxSurcharge := h.AntiSnipeStartTotalBps - h.BaseFeeBps
	surcharge := maxSurcharge * (h.AntiSnipeWindowSeconds - elapsed) / h.AntiSnipeWindowSeconds
	total := h.BaseFeeBps + surcharge
	if total > MaxTotalFeeBps {
		return MaxTotalFeeBps
	}
	return total
}

// quoteIsSpecified reports whether the swap's specified (input, exact-in) currency
// is the quote currency -- i.e. the swap is buying the launch token with the quote.
func (h *Hook) quoteIsSpecified(zeroForOne bool) bool {
	// exact-in: specified currency is currency0 iff zeroForOne.
	specifiedIsCurrency0 := zeroForOne
	quoteIsCurrency0 := !h.TokenIsCurrency0
	return specifiedIsCurrency0 == quoteIsCurrency0
}

// BeforeSwap only handles the exact-in (CalcOut) direction -- the only one
// pool_simulator.go's CalcAmountOut ever drives. When the quote currency is the
// swap's input, LaunchHook.sol charges the fee pre-swap (floor-rounded) so the
// AMM sees a reduced input; see AfterSwap for the output-side case.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !params.CalcOut {
		return nil, errCalcInUnsupported
	}
	if h.LaunchTime == 0 {
		// Never successfully Track()ed (RPC failure, or genuinely not yet registered
		// with the factory): a zero-value Extra has BaseFeeBps=0, which would
		// otherwise price this pool as fee-free instead of refusing to quote it.
		return nil, errNotTracked
	}
	if !h.quoteIsSpecified(params.ZeroForOne) {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
		}, nil
	}
	fee := feeAmount(params.AmountSpecified, h.totalFeeBps())
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   fee,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

// AfterSwap applies the fee post-swap (floor-rounded, on the realized output) when
// the quote currency is the swap's output side.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if !params.CalcOut {
		return nil, errCalcInUnsupported
	}
	if h.LaunchTime == 0 {
		return nil, errNotTracked
	}
	if h.quoteIsSpecified(params.ZeroForOne) {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}
	fee := feeAmount(params.AmountOut, h.totalFeeBps())
	return &uniswapv4.AfterSwapResult{HookFee: fee}, nil
}

// feeAmount mirrors LaunchHook.sol's _feeAmount for the exact-in (non-exactOutput)
// branch: floor(magnitude * feeBps / 10_000).
func feeAmount(magnitude *big.Int, feeBps int64) *big.Int {
	if feeBps == 0 || magnitude == nil || magnitude.Sign() == 0 {
		return bignumber.ZeroBI
	}
	return bignumber.MulDivDown(new(big.Int), magnitude, big.NewInt(feeBps), big.NewInt(bps))
}
