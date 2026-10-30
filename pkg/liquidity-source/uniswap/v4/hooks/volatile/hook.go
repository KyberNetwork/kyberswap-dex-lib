package volatile

import (
	"context"
	"math/big"
	"slices"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Extra is the hook's tracked state, read at one block together with the pool's slot0.
type Extra struct {
	Config FeeConfig `json:"c"`
	State  FeeState  `json:"s"`
}

// Hook embeds BaseHook by value, not the Hook interface, so a simulator survives the msgpack round trip
// (as pons-v2 and coocoo, #1800).
type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	Extra
	// MaxFee is the most this hook charges per side, in pips. The factory sets it from the hook address and it is never
	// tracked or stored in hX. 0, as in a Hook built without the factory, means Uniswap's 999,998.
	MaxFee uint32 `json:"-"`
	// advanced: State already holds this block's top-of-block advance (a swap of this sim ran).
	advanced bool
}

// SwapInfo carries the advanced fee state a first swap leaves behind, like the hook's on-chain write.
type SwapInfo struct {
	State FeeState
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: DexType}}
	if slices.Contains(ChangingFeeHookAddresses, param.HookAddress) {
		h.MaxFee = ChangingFeeMaxFee
	}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, append([]common.Address{HookAddress}, ChangingFeeHookAddresses...)...)

// maxFee is the ceiling of the fee this hook charges. ChangingFeeHook clamps the fee at 20,000 pips after the
// library's own clamp (ChangingFeeHook._chargedFee); Uniswap's hook is limited only by the library's 999,998.
func (h *Hook) maxFee() uint32 {
	if h.MaxFee == 0 {
		return maxFeeE6
	}
	return h.MaxFee
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	poolID := common.HexToHash(param.Pool.Address)
	target := hexutil.Encode(param.HookAddress[:])
	// ethrpc needs one struct target (abi tags) for a multi-output call; uintN decodes to *big.Int.
	var state struct {
		BlockNumber  *big.Int `abi:"blockNumber"` // unused: a sim swap always runs in a later block
		SqrtPriceTop *big.Int `abi:"sqrtPriceTopX96"`
		AskPermFee   *big.Int `abi:"askPermFeeE6"`
		TempFee      *big.Int `abi:"tempFeeE6"`
		TempOnBid    bool     `abi:"tempOnBid"`
	}
	var cfg struct {
		RestingSpread          *big.Int `abi:"restingSpreadE6"`
		BackrunCaptureFraction *big.Int `abi:"backrunCaptureFractionE6"`
		MidFadeRate            *big.Int `abi:"midFadeRateE6"`
		MaxMidFade             *big.Int `abi:"maxMidFadeE6"`
		SkewRate               *big.Int `abi:"skewRateE6"`
		MaxSkewStep            *big.Int `abi:"maxSkewStepE6"`
		MaxSkew                *big.Int `abi:"maxSkewE6"`
		BackrunThreshold       *big.Int `abi:"backrunThresholdE6"`
		RestoreFraction        *big.Int `abi:"restoreFractionE6"`
		RestoreThreshold       *big.Int `abi:"restoreThresholdE6"`
	}
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{ABI: volatilePairHookABI, Target: target, Method: "feeState",
			Params: []any{poolID}}, []any{&state}).
		AddCall(&ethrpc.Call{ABI: volatilePairHookABI, Target: target, Method: "feeConfig",
			Params: []any{poolID}}, []any{&cfg}).
		Aggregate(); err != nil {
		return nil, err
	}

	c := FeeConfig{
		RestingSpread: uint32(cfg.RestingSpread.Uint64()), BackrunCaptureFraction: uint32(cfg.BackrunCaptureFraction.Uint64()),
		MidFadeRate: uint32(cfg.MidFadeRate.Uint64()), MaxMidFade: uint32(cfg.MaxMidFade.Uint64()),
		SkewRate: uint32(cfg.SkewRate.Uint64()), MaxSkewStep: uint32(cfg.MaxSkewStep.Uint64()),
		MaxSkew: uint32(cfg.MaxSkew.Uint64()), BackrunThreshold: uint32(cfg.BackrunThreshold.Uint64()),
		RestoreFraction: uint32(cfg.RestoreFraction.Uint64()), RestoreThreshold: uint32(cfg.RestoreThreshold.Uint64()),
	}
	s := FeeState{AskPermFee: uint32(state.AskPermFee.Uint64()), TempFee: uint32(state.TempFee.Uint64()), TempOnBid: state.TempOnBid}
	s.SqrtPriceTop.SetFromBig(state.SqrtPriceTop)
	// The contract reverts getFee on an unset config; askPermFee <= restingSpread guards the unsigned subtractions.
	if c.RestingSpread == 0 || s.AskPermFee > c.RestingSpread {
		return nil, ErrInvalidFeeState
	}

	h.Extra = Extra{Config: c, State: s}
	return json.Marshal(h)
}

// BeforeSwap mirrors VolatilePairHook._beforeSwap. A sim swap runs in a later block than the tracked state,
// so the first one advances the state as the first swap of a new block does; later legs of the route
// (advanced) see that state and the post-swap price. The fee never depends on the swap size.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.Config.RestingSpread == 0 {
		return nil, ErrPoolIsNotTracked
	}
	cur := params.SqrtPriceX96
	if cur == nil || cur.IsZero() {
		return nil, ErrInvalidSqrtPrice
	}

	res := &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		Gas:              HookGas,
	}
	state := h.State
	if !h.advanced {
		h.Config.advance(&state, cur)
		res.SwapInfo = &SwapInfo{State: state}
	}
	res.SwapFee = uniswapv4.FeeAmount(min(h.Config.fee(&state, cur, params.ZeroForOne), h.maxFee()))
	return res, nil
}

func (h *Hook) UpdateBalance(swapInfo any) {
	if info, ok := swapInfo.(*SwapInfo); ok && info != nil {
		h.State = info.State
		h.advanced = true
	}
}

// CloneState deep-copies: every field is a value, so a struct copy is enough.
func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}
