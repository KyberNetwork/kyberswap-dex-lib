// Package fewshell implements Ring Protocol's FewV2ShellHook, a Uniswap v4 hook that fills every swap of its pools
// through a FewV2 pair instead of the pool's own liquidity. Source: Blockscout-verified FewV2ShellHook.sol and
// src/FewV2ShellHookBase.sol (solc 0.8.26), cited below as Base:<line>.
//
// A shell pool's own position is a decoy that never trades: beforeSwap + beforeSwapReturnDelta replace the whole
// swap. For exact input (the only kind a router executes) the hook:
//
//  1. checks the PoolManager holds amountIn of the input token (_requireSettlementInventory, Base:698),
//  2. wraps amountIn 1:1 into the input Few token and swaps it on the pool's registered FewV2 pair, which gives
//     fwOut = FewV2Math.getAmountOut (997/1000 fee),
//  3. checks the output Few token's contract holds fwOut of its underlying (_requireWrapperBacking, Base:713),
//  4. charges fee = ceil(fwOut*rate/1e6) of the output to the Uniswap fee controller's TOKEN_JAR, with rate = the
//     direction's slot0 protocol fee * 25 (_protocolFeeAmount, Base:488),
//  5. unwraps fwOut 1:1 and returns amountOut = fwOut - fee.
//
// A pool maps to one direct pair, fixed at beforeInitialize (Base:275) and readable with registeredRouteFor. Why a
// registration matters: an unregistered swap-permission hook falls to the dex-lib-private auto calibrator, which fits
// the decoy position and over-quotes small swaps by about 40%.
package fewshell

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Extra is the hook's tracked state (hX). Index i is the pool's currency i, whose Few token is Few[i]. The keys avoid
// the dex-lib-private calibrator's (f, b, lp, cl, u, k, v, a, r), so a pool still holding calibrator state reads as
// untracked (no Pair) until the first Track.
type Extra struct {
	// Pair is the registered FewV2 pair and Few its two tokens (registeredRouteFor); never changes after registration.
	Pair common.Address    `json:"pa"`
	Few  [2]common.Address `json:"fw"`
	// Reserves are the pair's reserves of Few[i]: the pair's token order is resolved at tracking.
	Reserves [2]uint256.Int `json:"rs"`
	// Inventory is the PoolManager's balance of currency i (the input must exist when the hook take()s it).
	Inventory [2]uint256.Int `json:"iv"`
	// Backing is the underlying balance of Few[i]'s contract (what an unwrap of Few[i] pays out of).
	Backing [2]uint256.Int `json:"bk"`
	Paused  bool           `json:"pz,omitempty"`
}

type Hook struct {
	// BaseHook is held by value, not as the Hook interface, so the msgpack-stored simulator round-trips the hook.
	uniswapv4.BaseHook `json:"-"`
	Extra

	// Rates is the protocol fee rate in ppm, [zeroForOne, oneForZero]: slot0's directional protocol fee times 25. It
	// comes from the pool's tracked pF at construction, not from hX.
	Rates [2]uint32 `json:"-"`
}

// SwapInfo is the state a swap leaves behind: the FewV2 pair and the wrappers move; the PoolManager's balances end
// where they began (the hook take()s the input and the router settles it, the hook settles the output and the
// router and fee jar take it).
type SwapInfo struct {
	Reserves [2]uint256.Int
	Backing  [2]uint256.Int
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddresses...)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	// Same exchange as the few wrapper hook (hooks/few/hook): the pools are listed by the uniswap-v4 lister.
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4}}
	if param == nil {
		return h
	}
	if err := param.HookExtra.Unmarshal(&h.Extra); err != nil {
		h.Extra = Extra{}
	}
	h.Rates = [2]uint32{
		uint32(uniswapv4.PoolProtocolFee(param.Pool, true)) * protocolFeeMultiplier,
		uint32(uniswapv4.PoolProtocolFee(param.Pool, false)) * protocolFeeMultiplier,
	}
	return h
}

// AllowEmptyTicks lets the v3 simulator take the zero amount BeforeSwap leaves it.
func (h *Hook) AllowEmptyTicks() bool { return true }

// TracksActivity: hX holds only chain state, so a change bumps Timestamp and BlockNumber.
func (h *Hook) TracksActivity() {}

// LivenessInterval makes the tracker stamp Extra "lv" with the time it last read the state: when the state changed,
// or when an unchanged state has stood trackInterval. "lv" lives outside hX, so hX keeps only chain state (see
// TracksActivity), and a quiet pool keeps identical Extra bytes between two stamps. NextTrackAt reads that stamp.
func (h *Hook) LivenessInterval() time.Duration { return trackInterval }

// NextTrackAt is the unix time the pool is tracked again with no event: one trackInterval after the last read, which
// "lv" holds (param.TrackedAt). The pair, the PoolManager's balances and the wrappers' backing move with trades that
// emit no event for the v4 pool, so without it a shell pool, which pool-service drops from the schedule while this
// returns 0, would wait for the next PoolManager event. A pool with no "lv" yet returns trackInterval seconds, a time
// long past, so it is tracked at once and then stamped. No RPC.
func (h *Hook) NextTrackAt(param *uniswapv4.HookParam) int64 {
	return param.TrackedAt + int64(trackInterval/time.Second)
}

// SqrtPriceLimit is the widest limit a router can pass. The hook rejects limit >= the FewV2 pair's price for
// zeroForOne (limit <= for oneForZero) and limit <= MIN_SQRT_PRICE (limit >= MAX), then checks the pair's price after
// the swap against it (Base:351-365). The decoy position's ticks, which the default limit comes from, say nothing about
// the pair's price, so the default could land on the wrong side of it. MIN+1 and MAX-1 always pass: the pair's price
// is inside [MIN, MAX) for any uint112 reserves (FewV2Price.validate), and no swap moves it past the limit.
func (h *Hook) SqrtPriceLimit(zeroForOne bool) *uint256.Int {
	if zeroForOne {
		return new(uint256.Int).AddUint64(big256.MinSqrtRatio, 1)
	}
	return new(uint256.Int).SubUint64(big256.MaxSqrtRatio, 1)
}

// Track reads the route (once), the pair and the balances the swap checks. Both multicalls are pinned to the tracker's
// block and carry its overrides.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.RpcClient == nil || param.Pool == nil || len(param.Pool.Tokens) != 2 {
		return nil, ErrPoolIsNotTracked
	}
	hook := hexutil.Encode(param.HookAddress[:])
	newRequest := func() *ethrpc.Request {
		return param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
			SetOverrides(param.Overrides)
	}

	// The route is immutable once registered, so it is read only while hX has none.
	if h.Pair == (common.Address{}) {
		var route struct {
			Few0 common.Address `abi:"few0"`
			Few1 common.Address `abi:"few1"`
			Pair common.Address `abi:"pair"`
		}
		if _, err := newRequest().AddCall(&ethrpc.Call{ABI: shellABI, Target: hook, Method: "registeredRouteFor",
			Params: []any{common.HexToHash(param.Pool.Address)}}, []any{&route}).Aggregate(); err != nil {
			return nil, fmt.Errorf("fewshell route: %w", err)
		}
		h.Pair, h.Few = route.Pair, [2]common.Address{route.Few0, route.Few1}
	}

	// A Few token's underlying is the pool's currency (checked by the hook's beforeInitialize), so its backing is the
	// currency's balance at the Few token. Native currencies do not exist here: useNativeCurrencyForWeth is false.
	currencies := [2]string{param.Pool.Tokens[0].Address, param.Pool.Tokens[1].Address}
	manager := uniswapv4.PoolManagerOf(param.Pool, param.Cfg.ChainID)
	var (
		token0   common.Address
		reserves struct {
			Reserve0, Reserve1 *big.Int
			BlockTimestampLast uint32
		}
		paused    bool
		inventory [2]*big.Int
		backing   [2]*big.Int
	)
	req := newRequest().
		AddCall(&ethrpc.Call{ABI: shellABI, Target: hexutil.Encode(h.Pair[:]), Method: "token0"}, []any{&token0}).
		AddCall(&ethrpc.Call{ABI: shellABI, Target: hexutil.Encode(h.Pair[:]), Method: "getReserves"}, []any{&reserves}).
		AddCall(&ethrpc.Call{ABI: shellABI, Target: hook, Method: "paused"}, []any{&paused})
	for i, currency := range currencies {
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: currency, Method: abi.Erc20BalanceOfMethod,
			Params: []any{manager}}, []any{&inventory[i]}).
			AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: currency, Method: abi.Erc20BalanceOfMethod,
				Params: []any{h.Few[i]}}, []any{&backing[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("fewshell state: %w", err)
	}

	// A Few token wraps its currency 1:1, so the pair's reserves of Few[i] are already in currency i's units.
	var res0, res1 = reserves.Reserve0, reserves.Reserve1
	switch token0 {
	case h.Few[0]:
	case h.Few[1]:
		res0, res1 = res1, res0
	default:
		return nil, ErrPairMismatch
	}
	next := Extra{Pair: h.Pair, Few: h.Few, Paused: paused}
	for dst, v := range map[*uint256.Int]*big.Int{
		&next.Reserves[0]: res0, &next.Reserves[1]: res1,
		&next.Inventory[0]: inventory[0], &next.Inventory[1]: inventory[1],
		&next.Backing[0]: backing[0], &next.Backing[1]: backing[1],
	} {
		if v == nil || dst.SetFromBig(v) {
			return nil, ErrBadBalance
		}
	}
	h.Extra = next
	return json.Marshal(&h.Extra)
}

// GetReserves is the hook's pseudoTotalValueLocked (Base:781-809): the pair's reserves of each currency, capped by
// what the Few token's contract can pay out and the PoolManager holds. The decoy position's own reserves are not
// liquidity a swap can use.
func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if h.Pair == (common.Address{}) {
		return nil, ErrPoolIsNotTracked
	}
	reserves := make(entity.PoolReserves, 2)
	for i := range reserves {
		amount := h.Reserves[i]
		if h.Reserves[1-i].IsZero() || amount.IsZero() {
			amount.Clear()
		} else {
			amount = *big256.Min(big256.Min(&amount, &h.Backing[i]), &h.Inventory[i])
		}
		reserves[i] = amount.Dec()
	}
	return reserves, nil
}

// BeforeSwap returns the whole swap: the input is consumed and the output supplied by the hook, leaving the v3
// simulator a zero amount. CalcIn reverses exact input, as the router only executes exact input; it does not
// quote FewV2ShellHookBase._swapExactOutput, which charges its fee on the input.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.Pair == (common.Address{}) {
		return nil, ErrPoolIsNotTracked
	} else if h.Paused {
		return nil, ErrPaused
	}
	// Zero reverts (InvalidSwapAmount) and so does an amount past int128 (SafeCast.toInt128, Base:465); the bound also
	// keeps every product below 2^256.
	amount, overflow := uint256.FromBig(params.AmountSpecified)
	if overflow || amount.IsZero() || amount.Cmp(big256.U2Pow127) >= 0 {
		return nil, ErrInvalidAmount
	}

	if params.CalcOut {
		info, out, err := h.swapExactIn(params.ZeroForOne, amount)
		if err != nil {
			return nil, err
		}
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   new(big.Int).Set(params.AmountSpecified),
			DeltaUnspecified: new(big.Int).Neg(out.ToBig()),
			Gas:              hookGas,
			SwapInfo:         info,
		}, nil
	}

	// The least input whose exact-in output is at least the amount asked: the pair must give the amount before the
	// protocol fee, so invert the fee first, then the pair.
	idxIn, idxOut := indexes(params.ZeroForOne)
	amountIn, err := minAmountIn(grossBeforeFee(amount, h.Rates[direction(params.ZeroForOne)]),
		&h.Reserves[idxIn], &h.Reserves[idxOut])
	if err != nil {
		return nil, err
	}
	info, _, err := h.swapExactIn(params.ZeroForOne, amountIn)
	if err != nil {
		return nil, err
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   new(big.Int).Neg(params.AmountSpecified),
		DeltaUnspecified: amountIn.ToBig(),
		Gas:              hookGas,
		SwapInfo:         info,
	}, nil
}

// swapExactIn is FewV2ShellHookBase._swapExactInput (Base:377-419) for amountIn of the input currency: it returns the
// state after the swap and the amount the trader receives, or the error of the revert the contract would hit.
func (h *Hook) swapExactIn(zeroForOne bool, amountIn *uint256.Int) (*SwapInfo, *uint256.Int, error) {
	idxIn, idxOut := indexes(zeroForOne)

	// amountSpecified.toInt128() (Base:465) reverts past int128: amountIn is below 2^127 from here on.
	if amountIn.Cmp(big256.U2Pow127) >= 0 {
		return nil, nil, ErrInvalidAmount
	}
	// Base:390: the PoolManager must hold the input the hook take()s.
	if h.Inventory[idxIn].Lt(amountIn) {
		return nil, nil, ErrInsufficientInventory
	}
	// Base:683 _hopState, then FewV2Math.getAmountOut.
	reserveIn, reserveOut := &h.Reserves[idxIn], &h.Reserves[idxOut]
	if reserveIn.IsZero() || reserveOut.IsZero() {
		return nil, nil, ErrDegeneratePair
	}
	fwOut, err := getAmountOut(amountIn, reserveIn, reserveOut)
	if err != nil {
		return nil, nil, err
	}
	// Base:392: the output wrapper must be able to pay out what the pair gave.
	if h.Backing[idxOut].Lt(fwOut) {
		return nil, nil, ErrInsufficientBacking
	}
	// Base:402-405: the protocol fee comes out of the output; a zero output reverts.
	amountOut := new(uint256.Int).Sub(fwOut, protocolFee(fwOut, h.Rates[direction(zeroForOne)]))
	if amountOut.IsZero() {
		return nil, nil, ErrInsufficientAmount
	}

	info := &SwapInfo{Reserves: h.Reserves, Backing: h.Backing}
	// The pair keeps its reserves in a uint112 (UniswapV2Pair._update).
	if info.Reserves[idxIn].Add(&info.Reserves[idxIn], amountIn).Gt(big256.UMaxU112) {
		return nil, nil, ErrReserveOverflow
	}
	info.Reserves[idxOut].Sub(&info.Reserves[idxOut], fwOut)
	info.Backing[idxIn].Add(&info.Backing[idxIn], amountIn)
	info.Backing[idxOut].Sub(&info.Backing[idxOut], fwOut)
	return info, amountOut, nil
}

func (h *Hook) UpdateBalance(swapInfo any) {
	if info, ok := swapInfo.(*SwapInfo); ok && info != nil {
		h.Reserves, h.Backing = info.Reserves, info.Backing
	}
}

// CloneState: all state is held by value.
func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

// indexes are the pool indexes of the input and the output currency.
func indexes(zeroForOne bool) (in, out int) {
	if zeroForOne {
		return 0, 1
	}
	return 1, 0
}

// direction indexes Hook.Rates: 0 for zeroForOne, 1 for oneForZero.
func direction(zeroForOne bool) int {
	if zeroForOne {
		return 0
	}
	return 1
}

var _ uniswapv4.Hook = (*Hook)(nil)
var _ uniswapv4.HookWithSchedule = (*Hook)(nil)
var _ uniswapv4.HookWithLiveness = (*Hook)(nil)
var _ uniswapv4.HookTracksActivity = (*Hook)(nil)
var _ uniswapv4.HookPriceLimitProvider = (*Hook)(nil)
