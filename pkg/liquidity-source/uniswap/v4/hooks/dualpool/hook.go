package dualpool

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook prices swaps against DualPoolHook pools. The pool's v4 tick state is
// always empty between swaps (the hook mints and burns its positions inside the
// swap), so the base simulator is told to allow empty ticks and the whole quote
// is produced here in BeforeSwap from the hook's effective reserves, its bucket
// distribution and slot0.
type Hook struct {
	uniswapv4.Hook
	hook  common.Address
	state Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4DualPool},
		hook: param.HookAddress,
	}
	_ = param.HookExtra.Unmarshal(&h.state)
	return h
}, HookAddresses...)

func (h *Hook) AllowEmptyTicks() bool { return true }

func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

func (h *Hook) CanAfterSwap(common.Address) bool { return false }

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	p := param.Pool
	if p == nil || len(p.Tokens) != 2 {
		return nil, ErrStateNotSet
	}
	var staticExtra uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	key := uniswapv4.PoolKey{
		Currency0:   currency(p.Tokens[0].Address, staticExtra.IsNative[0]),
		Currency1:   currency(p.Tokens[1].Address, staticExtra.IsNative[1]),
		Fee:         big.NewInt(int64(staticExtra.Fee)),
		TickSpacing: big.NewInt(int64(staticExtra.TickSpacing)),
		Hooks:       staticExtra.HooksAddress,
	}
	poolID := common.HexToHash(p.Address)

	stateView := defaultStateViewRobinhood
	if param.Cfg != nil && param.Cfg.StateViewAddress != "" {
		stateView = param.Cfg.StateViewAddress
	}

	var (
		balances     effectiveLiquidityResp
		distribution []distributionEntry
		live         bool
		slot0        slot0Resp
	)
	req := param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides)
	if param.BlockNumber != nil && param.BlockNumber.Sign() > 0 {
		req.SetBlockNumber(param.BlockNumber)
	}
	req.AddCall(&ethrpc.Call{
		ABI: dualPoolHookABI, Target: h.hook.Hex(), Method: "getEffectiveLiquidity", Params: []any{key},
	}, []any{&balances})
	req.AddCall(&ethrpc.Call{
		ABI: dualPoolHookABI, Target: h.hook.Hex(), Method: "getDistribution", Params: []any{poolID},
	}, []any{&distribution})
	req.AddCall(&ethrpc.Call{
		ABI: dualPoolHookABI, Target: h.hook.Hex(), Method: "livePools", Params: []any{poolID},
	}, []any{&live})
	req.AddCall(&ethrpc.Call{
		ABI: stateViewABI, Target: stateView, Method: "getSlot0", Params: []any{poolID},
	}, []any{&slot0})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	extra := Extra{
		Live:         live,
		Balance0:     uint256.MustFromBig(balances.Token0),
		Balance1:     uint256.MustFromBig(balances.Token1),
		SqrtPriceX96: uint256.MustFromBig(slot0.SqrtPriceX96),
		Tick:         int32(slot0.Tick.Int64()),
		ProtocolFee:  uint32(slot0.ProtocolFee.Uint64()),
		LpFee:        uint32(slot0.LpFee.Uint64()),
		Buckets:      make([]Bucket, 0, len(distribution)),
	}
	for _, d := range distribution {
		extra.Buckets = append(extra.Buckets, Bucket{
			TickLower: int32(d.TickLower.Int64()), TickUpper: int32(d.TickUpper.Int64()), WeightBps: d.WeightBps,
		})
	}
	h.state = extra
	return json.Marshal(extra)
}

// GetReserves reports the hook's effective reserves: what a swap can actually
// draw on. StateView ticks are empty for this pool, so the base estimate is not usable.
func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if h.state.Balance0 == nil || h.state.Balance1 == nil {
		return nil, nil
	}
	return entity.PoolReserves{h.state.Balance0.Dec(), h.state.Balance1.Dec()}, nil
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	s := &h.state
	if s.SqrtPriceX96 == nil || s.Balance0 == nil || s.Balance1 == nil {
		return nil, ErrStateNotSet
	}
	if !s.Live {
		return nil, ErrPoolNotLive
	}
	if len(s.Buckets) == 0 {
		return nil, ErrNoDistribution
	}
	if s.Balance0.IsZero() && s.Balance1.IsZero() {
		return nil, ErrNoReserves
	}
	amountSpecified, overflow := uint256.FromBig(params.AmountSpecified)
	if overflow || amountSpecified.IsZero() {
		return nil, lo.Ternary(params.CalcOut, uniswapv4.ErrInvalidAmountIn, uniswapv4.ErrInvalidAmountOut)
	}

	positions, err := allocate(s.Buckets, s.SqrtPriceX96, s.Balance0, s.Balance1)
	if err != nil {
		return nil, err
	}
	if len(positions) == 0 {
		return nil, ErrInsufficientLiquidity
	}
	fee := effectiveSwapFee(s.LpFee, s.ProtocolFee, params.ZeroForOne)
	swap := func(amountIn *uint256.Int) (*swapResult, error) {
		return swapExactIn(positions, s.SqrtPriceX96, int(s.Tick), params.ZeroForOne, amountIn, fee)
	}
	if !params.CalcOut {
		return h.calcIn(params, swap)
	}
	res, err := swap(amountSpecified)
	if err != nil {
		return nil, err
	}
	return h.beforeSwapResult(params.ZeroForOne, res,
		res.amountIn.ToBig(),                    // the whole input is consumed here
		new(big.Int).Neg(res.amountOut.ToBig()), // out -= (-amountOut)
	), nil
}

// calcIn reverses exact-in: the hook only prices exact-in, so it searches the min amountIn whose exact-in swap
// gives AmountSpecified out, and takes the whole swap off the pool.
func (h *Hook) calcIn(params *uniswapv4.BeforeSwapParams,
	swap func(amountIn *uint256.Int) (*swapResult, error)) (*uniswapv4.BeforeSwapResult, error) {
	var amountIn uint256.Int
	calcAmountOut := func(amountInBI *big.Int) (*big.Int, error) {
		if amountIn.SetFromBig(amountInBI) {
			return nil, uniswapv4.ErrInvalidAmountIn
		}
		res, err := swap(&amountIn)
		if err != nil {
			return nil, err
		}
		return res.amountOut.ToBig(), nil
	}
	// spot price guess: token1 per token0 = (sqrtPriceX96 / 2^96)^2
	var guess, price big.Int
	price.Mul(h.state.SqrtPriceX96.ToBig(), h.state.SqrtPriceX96.ToBig())
	if params.ZeroForOne {
		bignumber.MulDivUp(&guess, params.AmountSpecified, q192, &price)
	} else {
		bignumber.MulDivUp(&guess, params.AmountSpecified, &price, q192)
	}
	amountInBI, err := uniswapv4.MinAmountIn(params.AmountSpecified, &guess, calcAmountOut)
	if err != nil {
		return nil, err
	}
	amountIn.SetFromBig(amountInBI)
	res, err := swap(&amountIn)
	if err != nil {
		return nil, err
	}
	return h.beforeSwapResult(params.ZeroForOne, res,
		new(big.Int).Neg(params.AmountSpecified), // out += (-amountOut): nothing left for the pool
		res.amountIn.ToBig(),                     // in += amountIn
	), nil
}

func (h *Hook) beforeSwapResult(zeroForOne bool, res *swapResult, deltaSpecified,
	deltaUnspecified *big.Int) *uniswapv4.BeforeSwapResult {
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: deltaUnspecified,
		SwapFee:          uniswapv4.FeeAmount(h.state.LpFee),
		Gas:              swapGas,
		SwapInfo: SwapInfo{
			ZeroForOne:   zeroForOne,
			AmountIn:     res.amountIn,
			AmountOut:    res.amountOut,
			SqrtPriceX96: res.sqrtPriceX96,
			Tick:         int32(res.tick),
		},
	}
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	cloned.state.Balance0 = cloneU(h.state.Balance0)
	cloned.state.Balance1 = cloneU(h.state.Balance1)
	cloned.state.SqrtPriceX96 = cloneU(h.state.SqrtPriceX96)
	cloned.state.Buckets = append([]Bucket(nil), h.state.Buckets...)
	return &cloned
}

// UpdateBalance: the input (fee included) stays in the pool, the output leaves,
// and the pool price persists in slot0 between JIT cycles.
func (h *Hook) UpdateBalance(swapInfo any) {
	si, ok := swapInfo.(SwapInfo)
	if !ok || si.AmountIn == nil || si.AmountOut == nil {
		return
	}
	s := &h.state
	if si.ZeroForOne {
		s.Balance0 = new(uint256.Int).Add(s.Balance0, si.AmountIn)
		s.Balance1 = subFloor(s.Balance1, si.AmountOut)
	} else {
		s.Balance1 = new(uint256.Int).Add(s.Balance1, si.AmountIn)
		s.Balance0 = subFloor(s.Balance0, si.AmountOut)
	}
	if si.SqrtPriceX96 != nil {
		s.SqrtPriceX96 = si.SqrtPriceX96.Clone()
		s.Tick = si.Tick
	}
}

func currency(addr string, isNative bool) common.Address {
	if isNative {
		return common.Address{}
	}
	return common.HexToAddress(addr)
}

func cloneU(x *uint256.Int) *uint256.Int {
	if x == nil {
		return nil
	}
	return x.Clone()
}

func subFloor(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) < 0 {
		return new(uint256.Int)
	}
	return new(uint256.Int).Sub(a, b)
}
