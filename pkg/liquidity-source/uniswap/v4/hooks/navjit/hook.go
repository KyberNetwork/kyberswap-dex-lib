// Package navjit implements NavJitHook, the LOT/USDG venue hook of the MemeFiLP basket launchpad on
// Robinhood chain. A Lot (LOT-*) is an ERC-20 basket of tokenised stocks that anyone can mint from
// or redeem into its constituents, so its fair price is its NAV.
//
// The pool's own tick math does NOT describe a NavJit fill. Per swap, the hook:
//
//   - lifts its optional standing position out of the pool (no user ever trades against it);
//   - buy  (USDG in): buys the constituents in their USDG pools, mints exactly the LOT needed and
//     places it as a one-sided position starting at (actual cost + buySpreadBps);
//   - sell (LOT in):  places a one-sided USDG position starting at (NAV - sellSpreadBps);
//   - lets the swap fill against that position, then removes it, redeems/sells any LOT it holds,
//     requires the pool price to end within bandBps of NAV and the hook's USDG delta >= 0, and
//     puts the standing position back.
//
// The resulting quote is flat at ~cost+spread (buys) or ~NAV-spread (sells) and bends only with
// the constituent pools' own slippage. Porting that on-chain path would mean simulating every
// constituent pool, the basket and the NAV guard. Instead this plugin reads a LADDER of exact-in
// quotes from the stock V4Quoter on the pool in both directions at the tracked block, and BeforeSwap
// takes the whole specified amount and pays the linearly interpolated ladder output:
//
//   - the fill curve is close to linear and bends down with size (constituent slippage only
//     grows), so the chord between two rungs sits at or just below it. It is not exactly concave
//     (the mint rounds to whole constituent amounts, rippling it by ~±1 bp), so a chord can sit
//     up to ~1 bp above it; buys therefore lose buyHaircutPpm (1.5 bps) so they never over-quote;
//   - consecutive swaps in one direction on the same simulator state are priced as one walk
//     along the ladder (see Hook.consumedIn);
//   - below the first rung the quote is proportional to it (measured flat to < 0.01 bps between
//     $1 and $100 on all three live venues);
//   - above the top rung the quote is refused: beyond it the hook may revert OutsideBand,
//     SellExceedsFloat or a constituent pool may run dry, and the V4Quoter did not say;
//   - a sell (LOT in) above the PoolManager's LOT float is refused, mirroring SellExceedsFloat;
//   - with StaleCheck (route finding), a ladder older than maxAgeSec is refused.
//
// AfterSwap charges nothing (the hook returns a zero delta; its surplus is taken from its own
// USDG delta, not from the trader's), and the hook holds no return-delta permission at all.
package navjit

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Rung is one V4Quoter exact-input quote: In of the input currency returns Out of the other.
// Gas is the quoter's gasEstimate for the whole swap.
type Rung struct {
	In  *uint256.Int `json:"i"`
	Out *uint256.Int `json:"o"`
	Gas int64        `json:"g"`
}

type Extra struct {
	Tracked bool `json:"tr,omitempty"`
	// UsdgIs0: USDG is currency0, so a zeroForOne swap is a buy (USDG in, LOT out).
	UsdgIs0 bool `json:"u0,omitempty"`
	// Nav is NavGuard.checkedNavPerUnit18(lot) (USD per LOT, 1e18) at the tracked block; nil when
	// the guard reverts, and then both ladders are empty (the hook's swap path would revert too).
	Nav *uint256.Int `json:"n,omitempty"`
	// Float is the PoolManager's ERC-20 balance of the Lot: an UPPER bound on a LOT-in swap the
	// hook can redeem in-swap (see floatRungsBps for why the real cap is a little lower; the
	// ladder carries that).
	Float *uint256.Int `json:"f,omitempty"`
	// BandBps is getParams().bandBps: afterSwap reverts OutsideBand unless the pool ends within it
	// of NAV. UsdgDecimals converts NAV to the pool's sqrtPrice (see SqrtPriceLimit).
	BandBps      uint16 `json:"w,omitempty"`
	UsdgDecimals uint8  `json:"d,omitempty"`
	// Buy: USDG in -> LOT out. Sell: LOT in -> USDG out. Ascending in both In and Out.
	Buy  []Rung `json:"bl,omitempty"`
	Sell []Rung `json:"sl,omitempty"`
	// Block is the block every read of the last Track was pinned to; TrackedAt its unix time.
	Block     uint64 `json:"bn,omitempty"`
	TrackedAt int64  `json:"ts,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra

	// consumedIn/consumedOut accumulate, per ladder (dirBuy, dirSell), what earlier swaps on this
	// simulator state took, so the next swap in the same direction is priced further along the
	// fill curve: every buy pushes the constituent pools up, every sell pushes them down. The
	// opposite direction is left alone, which errs toward under-quoting it. Out is pre-haircut.
	consumedIn, consumedOut [2]uint256.Int
	staleCheck              bool
}

const (
	dirBuy  = 0
	dirSell = 1
)

// SwapInfo is what BeforeSwap hands to UpdateBalance: the input and the pre-haircut ladder
// output the swap consumed.
type SwapInfo struct {
	Dir       int
	AmountIn  uint256.Int
	RawAmount uint256.Int
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4NavJit},
		staleCheck: param.StaleCheck}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// AllowEmptyTicks: a venue with no standing position has no ticks at all, and BeforeSwap takes
// the whole amount, so the pool's own CL math always runs on a zero remainder.
func (h *Hook) AllowEmptyTicks() bool { return true }

// SqrtPriceLimit replaces the tick-derived limit, which sits at the standing position's edge and
// stops the JIT fill part-way. It is the band edge afterSwap enforces (|px - nav| * BPS <=
// nav * bandBps) on the side this swap moves toward, pulled in by priceLimitBufferBps and rounded
// toward NAV, so the limit never lies past the band. nil (untracked) keeps the default.
func (h *Hook) SqrtPriceLimit(zeroForOne bool) *uint256.Int {
	if h.Nav == nil || h.BandBps <= priceLimitBufferBps || h.UsdgDecimals > 18 {
		return nil
	}
	var usdPerLot, factor uint256.Int // USD per LOT, 1e18, at the limit
	off := uint64(h.BandBps - priceLimitBufferBps)
	if zeroForOne == h.UsdgIs0 { // a buy pushes USD per LOT up
		u256.MulDivDown(&usdPerLot, h.Nav, factor.SetUint64(bps+off), u256.UBasisPoint)
	} else {
		u256.MulDivUp(&usdPerLot, h.Nav, factor.SetUint64(bps-off), u256.UBasisPoint)
	}
	// NavJitHook._sqrtPriceForUsd18: ratioX192 = scale<<192/p (USDG is currency0) or p<<192/scale.
	scale := u256.TenPow(36 - int(h.UsdgDecimals))
	num, den := scale, &usdPerLot
	if !h.UsdgIs0 {
		num, den = &usdPerLot, scale
	}
	var ratio uint256.Int
	limit := new(uint256.Int)
	if !zeroForOne { // an upper bound: floor
		return limit.Sqrt(u256.MulDivDown(&ratio, num, u256.U2Pow192, den))
	}
	// a lower bound: ceil
	limit.Sqrt(u256.MulDivUp(&ratio, num, u256.U2Pow192, den))
	if factor.Mul(limit, limit).Lt(&ratio) {
		limit.AddUint64(limit, 1)
	}
	return limit
}

func (h *Hook) CloneState() uniswapv4.Hook {
	// The ladder is replaced wholesale by Track and never written by a swap; the consumed amounts
	// are value arrays, so the shallow copy already owns them.
	cloned := *h
	return &cloned
}

func (h *Hook) UpdateBalance(swapInfo any) {
	info, ok := swapInfo.(*SwapInfo)
	if !ok || info == nil {
		return
	}
	h.consumedIn[info.Dir].Add(&h.consumedIn[info.Dir], &info.AmountIn)
	h.consumedOut[info.Dir].Add(&h.consumedOut[info.Dir], &info.RawAmount)
}

// GetReserves reports what the venue can actually deliver: the top rung's output in each
// direction (USDG out of the largest quotable sell, LOT out of the largest quotable buy).
func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if !h.Tracked {
		return nil, nil
	}
	usdgRes, lotRes := topOut(h.Sell, false), topOut(h.Buy, true)
	if h.UsdgIs0 {
		return entity.PoolReserves{usdgRes, lotRes}, nil
	}
	return entity.PoolReserves{lotRes, usdgRes}, nil
}

func topOut(l []Rung, buy bool) string {
	if len(l) == 0 {
		return "0"
	}
	if buy {
		var out uint256.Int
		return haircut(&out, l[len(l)-1].Out).Dec()
	}
	return l[len(l)-1].Out.Dec()
}

// hookParams mirrors NavJitHook.Params; only BandBps is used (SqrtPriceLimit).
type hookParams struct {
	BuySpreadBps  uint16
	SellSpreadBps uint16
	BandBps       uint16
	SizeBufferBps uint16
	WidthTicks    *big.Int
}

type quoteParams struct {
	PoolKey struct {
		Currency0   common.Address
		Currency1   common.Address
		Fee         *big.Int
		TickSpacing *big.Int
		Hooks       common.Address
	}
	ZeroForOne  bool
	ExactAmount *big.Int
	HookData    []byte
}

type quoteResult struct {
	AmountOut   *big.Int
	GasEstimate *big.Int
}

// Track reads, all pinned to one block: the hook's USDG, NAV guard and PoolManager and which
// currency the pool is the venue of; then NAV, the PoolManager's LOT float and the buy ladder;
// then the sell ladder, sized at NAV. On Robinhood, pool-service's multicall is ArbMulticall2, so
// HookParam.BlockNumber is the L2 block (plain Multicall3 would report the L1 one).
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	p := param.Pool
	if p == nil || len(p.Tokens) < 2 {
		return nil, ErrNotAVenue
	}
	quoter, ok := QuoterByChain[param.Cfg.ChainID]
	if !ok {
		return nil, ErrPoolIsNotTracked
	}
	var staticExtra uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	hookAddr := hexutil.Encode(param.HookAddress[:])
	poolID := common.HexToHash(p.Address)
	tok0, tok1 := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)

	// Round 1: the hook surface and which side is the Lot.
	var (
		usdg, navGuard, pm common.Address
		venue0, venue1     common.Hash
		params             struct{ P hookParams } // getParams returns one tuple
	)
	req := param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}
	res, err := req.
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "usdg"}, []any{&usdg}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "navGuard"}, []any{&navGuard}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "poolManager"}, []any{&pm}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "getParams"}, []any{&params}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "venueOf", Params: []any{tok0}}, []any{&venue0}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "venueOf", Params: []any{tok1}}, []any{&venue1}).
		Aggregate()
	if err != nil {
		return nil, err
	}
	block := param.BlockNumber
	if block == nil {
		block = res.BlockNumber
	}

	var usdgIs0 bool
	var lot common.Address
	switch {
	case venue1 == poolID && usdg == tok0:
		usdgIs0, lot = true, tok1
	case venue0 == poolID && usdg == tok1:
		usdgIs0, lot = false, tok0
	default:
		return nil, ErrNotAVenue
	}
	usdgIdx := 1
	if usdgIs0 {
		usdgIdx = 0
	}
	usdgDecimals := p.Tokens[usdgIdx].Decimals

	key := quoteParams{HookData: []byte{}}
	key.PoolKey.Currency0, key.PoolKey.Currency1, key.PoolKey.Hooks = tok0, tok1, param.HookAddress
	key.PoolKey.Fee = big.NewInt(int64(staticExtra.Fee))
	key.PoolKey.TickSpacing = big.NewInt(int64(staticExtra.TickSpacing))

	// Round 2: NAV, float and the buy ladder (USDG in).
	var nav, float *big.Int
	usdgUnit := bignumber.TenPowInt(usdgDecimals)
	buyIn := make([]*big.Int, len(rungsUsd))
	for i, usd := range rungsUsd {
		buyIn[i] = new(big.Int).Mul(new(big.Int).SetUint64(usd), usdgUnit)
	}
	buyOut := make([]quoteResult, len(buyIn))
	req = param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides).SetBlockNumber(block).
		AddCall(&ethrpc.Call{ABI: navGuardABI, Target: hexutil.Encode(navGuard[:]),
			Method: "checkedNavPerUnit18", Params: []any{lot}}, []any{&nav}).
		AddCall(&ethrpc.Call{ABI: erc20ABI, Target: hexutil.Encode(lot[:]),
			Method: "balanceOf", Params: []any{pm}}, []any{&float})
	addLadder(req, quoter, key, usdgIs0, buyIn, buyOut)
	res2, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	h.Extra = Extra{Tracked: true, UsdgIs0: usdgIs0, BandBps: params.P.BandBps, UsdgDecimals: usdgDecimals,
		Block: block.Uint64(), TrackedAt: time.Now().Unix()}
	// The hook's beforeSwap reads the same guard: if it reverts, so does every swap.
	if !res2.Result[0] || nav == nil || nav.Sign() == 0 || !res2.Result[1] || float == nil {
		return json.Marshal(h)
	}
	h.Nav, h.Float = uint256.MustFromBig(nav), uint256.MustFromBig(float)
	h.Buy = toLadder(buyIn, buyOut, res2.Result[2:])

	// Round 3: the sell ladder (LOT in), sized at NAV, plus rungs just under the float.
	sellIn := make([]*big.Int, 0, len(rungsUsd)+len(floatRungsBps))
	for _, usd := range rungsUsd {
		amt := new(big.Int).Mul(new(big.Int).SetUint64(usd), bignumber.TenPowInt(36))
		if amt.Div(amt, nav).Cmp(float) >= 0 {
			break
		}
		sellIn = append(sellIn, amt)
	}
	for _, fb := range floatRungsBps {
		fr := new(big.Int).Mul(float, big.NewInt(fb))
		if fr.Div(fr, bignumber.BasisPoint).Sign() > 0 && (len(sellIn) == 0 || fr.Cmp(sellIn[len(sellIn)-1]) > 0) {
			sellIn = append(sellIn, fr)
		}
	}
	sellOut := make([]quoteResult, len(sellIn))
	req = param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides).SetBlockNumber(block)
	addLadder(req, quoter, key, !usdgIs0, sellIn, sellOut)
	res3, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	h.Sell = toLadder(sellIn, sellOut, res3.Result)
	return json.Marshal(h)
}

func addLadder(req *ethrpc.Request, quoter common.Address, key quoteParams, zeroForOne bool,
	amounts []*big.Int, out []quoteResult) {
	target := hexutil.Encode(quoter[:])
	for i, amt := range amounts {
		qp := key
		qp.ZeroForOne = zeroForOne
		qp.ExactAmount = amt
		req.AddCall(&ethrpc.Call{ABI: quoterABI, Target: target, Method: "quoteExactInputSingle",
			Params: []any{qp}}, []any{&out[i]})
	}
}

// toLadder keeps the contiguous run of successful, strictly increasing rungs from the bottom:
// a rung that reverted ends the ladder, since nothing above it is known to fill.
func toLadder(in []*big.Int, out []quoteResult, ok []bool) []Rung {
	ladder := make([]Rung, 0, len(in))
	for i := range in {
		if i >= len(ok) || !ok[i] || out[i].AmountOut == nil || out[i].AmountOut.Sign() <= 0 {
			break
		}
		r := Rung{In: uint256.MustFromBig(in[i]), Out: uint256.MustFromBig(out[i].AmountOut)}
		if out[i].GasEstimate != nil && out[i].GasEstimate.IsInt64() {
			r.Gas = out[i].GasEstimate.Int64()
		}
		if n := len(ladder); n > 0 && (!r.In.Gt(ladder[n-1].In) || !r.Out.Gt(ladder[n-1].Out)) {
			break
		}
		ladder = append(ladder, r)
	}
	return ladder
}

// BeforeSwap takes the whole specified amount and pays the interpolated ladder output, so the
// pool's own CL math runs on zero. CalcOut (exact-in): in -= specified, out -= unspecified, hence
// DeltaSpecified = amountIn and DeltaUnspecified = -amountOut. CalcIn (the reverse of exact-in):
// out += specified, in += unspecified, hence DeltaSpecified = -amountOut, DeltaUnspecified = amountIn.
//
// Both are priced as the increment of the ladder between what earlier swaps on this state
// consumed and that plus this swap (see consumedIn); buys then lose buyHaircutPpm.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.Tracked {
		return nil, ErrPoolIsNotTracked
	} else if h.staleCheck && time.Now().Unix()-h.TrackedAt > maxAgeSec {
		return nil, ErrStale
	}
	dir, ladder := dirSell, h.Sell
	if params.ZeroForOne == h.UsdgIs0 {
		dir, ladder = dirBuy, h.Buy
	}
	if len(ladder) == 0 {
		return nil, ErrEmptyLadder
	}
	var amt uint256.Int
	if params.AmountSpecified.Sign() <= 0 || amt.SetFromBig(params.AmountSpecified) {
		return nil, ErrInvalidAmount
	}
	usedIn, usedOut := &h.consumedIn[dir], &h.consumedOut[dir]
	info := &SwapInfo{Dir: dir}

	if params.CalcOut {
		var totalIn uint256.Int
		if _, of := totalIn.AddOverflow(usedIn, &amt); of {
			return nil, ErrBeyondLadder
		}
		if dir == dirSell && h.Float != nil && totalIn.Gt(h.Float) {
			return nil, ErrSellExceedsFloat
		}
		totalOut, gas, err := quoteOut(ladder, &totalIn)
		if err != nil {
			return nil, err
		}
		if !totalOut.Gt(usedOut) {
			return nil, ErrZeroOutput
		}
		info.AmountIn.Set(&amt)
		out := info.RawAmount.Sub(&totalOut, usedOut)
		if dir == dirBuy {
			out = haircut(&totalOut, out) // totalOut is no longer needed
		}
		if out.IsZero() {
			return nil, ErrZeroOutput
		}
		deltaUnspecified := out.ToBig()
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   new(big.Int).Set(params.AmountSpecified),
			DeltaUnspecified: deltaUnspecified.Neg(deltaUnspecified),
			Gas:              hookGas(gas),
			SwapInfo:         info,
		}, nil
	}

	// Exact-out: the ladder must deliver enough raw output that the haircut leaves amt.
	raw := info.RawAmount.Set(&amt)
	if dir == dirBuy {
		u256.MulDivUp(raw, &amt, uPpm, uPpmAfterCut)
	}
	var totalOut uint256.Int
	if _, of := totalOut.AddOverflow(usedOut, raw); of {
		return nil, ErrBeyondLadder
	}
	totalIn, gas, err := quoteIn(ladder, &totalOut)
	if err != nil {
		return nil, err
	}
	if !totalIn.Gt(usedIn) {
		return nil, ErrZeroOutput
	}
	if dir == dirSell && h.Float != nil && totalIn.Gt(h.Float) {
		return nil, ErrSellExceedsFloat
	}
	info.AmountIn.Sub(&totalIn, usedIn)
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   new(big.Int).Neg(params.AmountSpecified),
		DeltaUnspecified: info.AmountIn.ToBig(),
		Gas:              hookGas(gas),
		SwapInfo:         info,
	}, nil
}

var (
	uPpm         = uint256.NewInt(ppm)
	uPpmAfterCut = uint256.NewInt(ppm - buyHaircutPpm)
)

// haircut sets z = floor(x * (ppm - buyHaircutPpm) / ppm) and returns z.
func haircut(z, x *uint256.Int) *uint256.Int {
	return u256.MulDivDown(z, x, uPpmAfterCut, uPpm)
}

// hookGas: the quoter's estimate covers the whole swap; the v4 simulator adds its own base gas.
func hookGas(quoterGas int64) int64 {
	return max(quoterGas-uniswapv4.DefaultGas.BaseGas, 0)
}

// quoteOut interpolates the exact-in ladder at amountIn, rounding down.
func quoteOut(ladder []Rung, amountIn *uint256.Int) (out uint256.Int, gas int64, err error) {
	if amountIn.Gt(ladder[len(ladder)-1].In) {
		return out, 0, ErrBeyondLadder
	}
	i := 0
	for amountIn.Gt(ladder[i].In) {
		i++
	}
	hi := ladder[i]
	if i == 0 {
		u256.MulDivDown(&out, amountIn, hi.Out, hi.In)
	} else {
		lo := ladder[i-1]
		var dx, dy, span uint256.Int
		dx.Sub(amountIn, lo.In)
		dy.Sub(hi.Out, lo.Out)
		span.Sub(hi.In, lo.In)
		u256.MulDivDown(&out, &dx, &dy, &span).Add(&out, lo.Out)
	}
	if out.IsZero() {
		return out, 0, ErrZeroOutput
	}
	return out, hi.Gas, nil
}

// quoteIn inverts the same chords at amountOut, rounding the input up.
func quoteIn(ladder []Rung, amountOut *uint256.Int) (in uint256.Int, gas int64, err error) {
	if amountOut.Gt(ladder[len(ladder)-1].Out) {
		return in, 0, ErrBeyondLadder
	}
	i := 0
	for amountOut.Gt(ladder[i].Out) {
		i++
	}
	hi := ladder[i]
	if i == 0 {
		u256.MulDivUp(&in, amountOut, hi.In, hi.Out)
	} else {
		lo := ladder[i-1]
		var dy, dx, span uint256.Int
		dy.Sub(amountOut, lo.Out)
		dx.Sub(hi.In, lo.In)
		span.Sub(hi.Out, lo.Out)
		u256.MulDivUp(&in, &dy, &dx, &span).Add(&in, lo.In)
	}
	if in.IsZero() {
		return in, 0, ErrZeroOutput
	}
	return in, hi.Gas, nil
}
