// Package arcade implements the Uniswap v4 hook of the Arcade launchpad on Arc
// (ArcadeHook.sol, 0x695cfF9C7F11fa87ca05c7b0A0fa64C3554B3eCe). One hook serves every
// launch; what a swap pays depends on the pool's launch mode and lifecycle:
//
//   - PUMP (bonding-curve launch): while Curving the pool holds no liquidity and every
//     V4 swap reverts (trades go through the hook's own buy/sell), as do swaps during the
//     single graduation transaction. Once Graduated the pool's LP fee is 0 and the hook
//     takes the whole trading fee in USDC: on the input when USDC is sold (beforeSwap,
//     specified delta) and on the output when USDC is bought (afterSwap, unspecified
//     delta). The fee decays linearly in log-mcap from 1% at graduation to 0.30%, read
//     from a stored EMA oracle that a swap never moves before paying.
//   - CLANKER / RWA (direct launch, graduated from birth): the fee is the pool's native
//     static LP fee (already in the PoolKey), the hook takes nothing, and a buy whose
//     token output tops a per-transaction cap reverts during the first five minutes.
//
// The curve-phase anti-snipe skim is inert once a pool is graduated
// (ArcadeHook._currentSnipeBps returns 0), so no graduated swap ever pays it.
package arcade

import (
	"context"
	"errors"
	"math/big"
	"strings"
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
	ErrNotTracked        = errors.New("arcade: pool state not tracked yet, refusing to quote at an unknown fee")
	ErrNotGraduated      = errors.New("arcade: pool is on its bonding curve or graduating, V4 swaps revert")
	ErrUnknownMode       = errors.New("arcade: unsupported launch mode")
	ErrBuyExceedsCap     = errors.New("arcade: buy exceeds the per-transaction cap of the launch window")
	ErrCalcInUnsupported = errors.New("arcade: exact-out not supported")
)

// NowFn is a var so tests can pin the clock used by the launch-window buy cap.
var NowFn = func() int64 { return time.Now().Unix() }

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Arcade}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// Extra is the swap-relevant hook state for one pool.
type Extra struct {
	Tracked bool  `json:"tr,omitempty"`
	Mode    uint8 `json:"m,omitempty"`
	Status  uint8 `json:"s,omitempty"`

	// PUMP fee oracle (ArcadeHook.feeObs) and USDC's side of the pool.
	UsdcIsCurrency0 bool  `json:"u0,omitempty"`
	ObsInit         bool  `json:"oi,omitempty"`
	EmaTickE3       int64 `json:"ema,omitempty"`
	GradMcapTick    int64 `json:"gt,omitempty"`

	// CLANKER / RWA per-transaction buy cap: the quote asset's side, the launch time
	// and whether the owner left the cap enabled (clankerMaxBuyBps != 0).
	QuoteIsCurrency0 bool  `json:"q0,omitempty"`
	LaunchedAt       int64 `json:"la,omitempty"`
	BuyCapEnabled    bool  `json:"bc,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

// Track re-reads the pool's lifecycle, mode, fee oracle and buy-cap inputs. The oracle
// moves on every graduated swap (at most once per block timestamp), so nothing here is
// cached across tracker passes.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hook := hexutil.Encode(param.HookAddress[:])
	poolID := common.HexToHash(param.Pool.Address)
	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)
	token1 := common.HexToAddress(param.Pool.Tokens[1].Address)

	var (
		state     curveStateRaw
		obs       feeObsRaw
		usdc      common.Address
		reg0      bool
		maxBuyBps uint16
	)
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{ABI: ArcadeHookABI, Target: hook, Method: "curveStates", Params: []any{poolID}}, []any{&state}).
		AddCall(&ethrpc.Call{ABI: ArcadeHookABI, Target: hook, Method: "feeObs", Params: []any{poolID}}, []any{&obs}).
		AddCall(&ethrpc.Call{ABI: ArcadeHookABI, Target: hook, Method: "USDC"}, []any{&usdc}).
		AddCall(&ethrpc.Call{ABI: ArcadeHookABI, Target: hook, Method: "registeredLaunches", Params: []any{token0}}, []any{&reg0}).
		AddCall(&ethrpc.Call{ABI: ArcadeHookABI, Target: hook, Method: "clankerMaxBuyBps"}, []any{&maxBuyBps}).
		Aggregate(); err != nil {
		return nil, err
	}

	// The launch token is the registered side; the quote is the other one (USDC, or
	// quoteAssetOf(token) for ARCADE-paired CLANKER and RWA launches).
	launchToken := token1
	if reg0 {
		launchToken = token0
	}
	var (
		quote common.Address
		pos   clankerPosRaw
	)
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{ABI: ArcadeHookABI, Target: hook, Method: "quoteAssetOf", Params: []any{launchToken}}, []any{&quote}).
		AddCall(&ethrpc.Call{ABI: ArcadeHookABI, Target: hook, Method: "clankerPos", Params: []any{launchToken}}, []any{&pos}).
		Aggregate(); err != nil {
		return nil, err
	}
	if quote == (common.Address{}) {
		quote = usdc
	}

	h.Extra = Extra{
		Tracked:          true,
		Mode:             state.Mode,
		Status:           state.Status,
		UsdcIsCurrency0:  sameAddress(token0, usdc),
		ObsInit:          obs.Init,
		EmaTickE3:        obs.EmaTickE3,
		QuoteIsCurrency0: sameAddress(token0, quote),
		LaunchedAt:       int64(pos.LaunchedAt),
		BuyCapEnabled:    maxBuyBps != 0,
	}
	if obs.GradMcapTick != nil {
		h.GradMcapTick = obs.GradMcapTick.Int64()
	}
	return json.Marshal(h)
}

func sameAddress(a, b common.Address) bool {
	return strings.EqualFold(a.Hex(), b.Hex())
}

// PumpFeeBps mirrors ArcadeHook._feeBps for a PUMP pool.
func (e *Extra) PumpFeeBps() int64 {
	if !e.ObsInit {
		return PumpFeeMaxBps
	}
	// Solidity int256 division truncates toward zero, as Go's does.
	growth := e.EmaTickE3/1_000 - e.GradMcapTick
	if growth <= 0 {
		return PumpFeeMaxBps
	}
	if growth >= PumpFeeFloorTicks {
		return PumpFeeMinBps
	}
	drop := (int64(PumpFeeMaxBps-PumpFeeMinBps) * growth) / PumpFeeFloorTicks
	return PumpFeeMaxBps - drop
}

// PerTxMaxBuyTokens mirrors ArcadeHook._perTxMaxBuyTokens. Returns nil when uncapped.
func (e *Extra) PerTxMaxBuyTokens(now int64) *big.Int {
	elapsed := now - e.LaunchedAt
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed >= buyCapWindowSeconds {
		return nil
	}
	capBps := int64(100 + (elapsed/buyCapStepSeconds)*100)
	return bignumber.MulDivDown(new(big.Int), TotalSupply, big.NewInt(capBps), big.NewInt(bps))
}

func (e *Extra) check(calcOut bool) error {
	if !calcOut {
		return ErrCalcInUnsupported
	}
	if !e.Tracked {
		return ErrNotTracked
	}
	if e.Status != StatusGraduated {
		return ErrNotGraduated
	}
	switch e.Mode {
	case ModePump, ModeClanker, ModeRwa:
		return nil
	default:
		return ErrUnknownMode
	}
}

// fee is floor(amount * feeBps / 10_000), after ArcadeHook's MAX_TOTAL_TAKE_BPS clamp.
func fee(amount *big.Int, feeBps int64) *big.Int {
	if feeBps > MaxTotalTakeBps {
		feeBps = MaxTotalTakeBps
	}
	if feeBps <= 0 || amount == nil || amount.Sign() == 0 {
		return bignumber.ZeroBI
	}
	return bignumber.MulDivDown(new(big.Int), amount, big.NewInt(feeBps), big.NewInt(bps))
}

// BeforeSwap (exact-in): a graduated PUMP pool takes its fee from the specified input
// when that input is USDC. Everything else passes through untouched.
func (e *Extra) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if err := e.check(params.CalcOut); err != nil {
		return nil, err
	}
	result := &uniswapv4.BeforeSwapResult{DeltaSpecified: bignumber.ZeroBI, DeltaUnspecified: bignumber.ZeroBI}
	// exact-in: the specified currency is currency0 iff zeroForOne.
	if e.Mode == ModePump && params.ZeroForOne == e.UsdcIsCurrency0 {
		result.DeltaSpecified = fee(params.AmountSpecified, e.PumpFeeBps())
	}
	return result, nil
}

// AfterSwap (exact-in): a graduated PUMP pool takes its fee from the unspecified USDC
// output; a CLANKER / RWA buy reverts above the launch-window per-transaction cap.
func (e *Extra) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if err := e.check(params.CalcOut); err != nil {
		return nil, err
	}
	if e.Mode == ModePump {
		if params.ZeroForOne != e.UsdcIsCurrency0 { // selling the token for USDC
			return &uniswapv4.AfterSwapResult{HookFee: fee(params.AmountOut, e.PumpFeeBps())}, nil
		}
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}
	isBuy := params.ZeroForOne == e.QuoteIsCurrency0 // quote in, launch token out
	if isBuy && e.BuyCapEnabled {
		if limit := e.PerTxMaxBuyTokens(NowFn()); limit != nil && params.AmountOut.Cmp(limit) > 0 {
			return nil, ErrBuyExceedsCap
		}
	}
	return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
}

// Delegate rather than rely on promotion: Hook embeds both the uniswapv4.Hook interface
// and Extra at the same depth, both with BeforeSwap/AfterSwap.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return h.Extra.BeforeSwap(params)
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return h.Extra.AfterSwap(params)
}
