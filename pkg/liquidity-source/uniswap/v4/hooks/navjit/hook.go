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
//     (the mint rounds to whole constituent amounts): the largest over-quote measured live on
//     2026-10-01 was +0.15 bps, the largest under-quote -3.9 bps before the ladder was densified;
//   - below the first rung the quote is proportional to it (measured flat to < 0.01 bps between
//     $1 and $100 on all three live venues);
//   - above the top rung the quote is refused: beyond it the hook may revert OutsideBand,
//     SellExceedsFloat or a constituent pool may run dry, and the V4Quoter did not say;
//   - a sell (LOT in) above the PoolManager's LOT float is refused, mirroring SellExceedsFloat.
//
// AfterSwap charges nothing (the hook returns a zero delta; its surplus is taken from its own
// USDG delta, not from the trader's), and the hook holds no return-delta permission at all.
package navjit

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
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

// Params mirrors NavJitHook.Params. Informational: the ladder already prices them in.
type Params struct {
	BuySpreadBps  uint16   `json:"b"`
	SellSpreadBps uint16   `json:"s"`
	BandBps       uint16   `json:"w"`
	SizeBufferBps uint16   `json:"z"`
	WidthTicks    *big.Int `json:"t"`
}

type Extra struct {
	Tracked bool `json:"tr,omitempty"`
	// UsdgIs0: USDG is currency0, so a zeroForOne swap is a buy (USDG in, LOT out).
	UsdgIs0 bool `json:"u0,omitempty"`
	// Nav is NavGuard.checkedNavPerUnit18(lot) (USD per LOT, 1e18) at the tracked block; nil if
	// the hook exposes no NAV guard.
	Nav *uint256.Int `json:"n,omitempty"`
	// Float is the PoolManager's ERC-20 balance of the Lot: an UPPER bound on a LOT-in swap the
	// hook can redeem in-swap (see floatRungsBps for why the real cap is a little lower; the
	// ladder carries that). nil if unread, and then only the ladder bounds sells.
	Float  *uint256.Int `json:"f,omitempty"`
	Params *Params      `json:"p,omitempty"`
	// Buy: USDG in -> LOT out. Sell: LOT in -> USDG out. Ascending in both In and Out.
	Buy  []Rung `json:"bl,omitempty"`
	Sell []Rung `json:"sl,omitempty"`
	// Block is the block every read of the last Track was pinned to.
	Block uint64 `json:"bn,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4NavJit}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// AllowEmptyTicks: a venue with no standing position has no ticks at all, and BeforeSwap takes
// the whole amount, so the pool's own CL math always runs on a zero remainder.
func (h *Hook) AllowEmptyTicks() bool { return true }

func (h *Hook) CloneState() uniswapv4.Hook {
	// The ladder is replaced wholesale by Track and never written by a swap.
	cloned := *h
	return &cloned
}

// GetReserves reports what the venue can actually deliver: the top rung's output in each
// direction (USDG out of the largest quotable sell, LOT out of the largest quotable buy).
func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if !h.Tracked {
		return nil, nil
	}
	usdgRes, lotRes := topOut(h.Sell), topOut(h.Buy)
	if h.UsdgIs0 {
		return entity.PoolReserves{usdgRes, lotRes}, nil
	}
	return entity.PoolReserves{lotRes, usdgRes}, nil
}

func topOut(l []Rung) string {
	if len(l) == 0 {
		return "0"
	}
	return l[len(l)-1].Out.Dec()
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

// Track reads, all pinned to one block: the hook's USDG, NAV guard, PoolManager and params; which
// currency the pool is the venue of; NAV and the PoolManager's LOT float; then V4Quoter exact-in
// ladders in both directions. Only the ladder is load-bearing: every other read is tolerated
// missing, so a later hook generation with a different admin surface still quotes.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.RpcClient == nil {
		if !h.Tracked {
			return nil, ErrPoolIsNotTracked
		}
		return json.Marshal(h)
	}
	p := param.Pool
	if p == nil || len(p.Tokens) < 2 {
		return nil, ErrNotAVenue
	}
	var staticExtra uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	chainID := valueobject.ChainIDRobinhood
	if param.Cfg != nil && param.Cfg.ChainID != 0 {
		chainID = param.Cfg.ChainID
	}
	quoter, ok := QuoterByChain[chainID]
	if !ok {
		return nil, ErrPoolIsNotTracked
	}

	hookAddr := hexutil.Encode(param.HookAddress[:])
	poolID := common.HexToHash(p.Address)
	tok0, tok1 := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)

	// Round 1: hook surface + which side is the Lot. On an Arbitrum-stack chain it is NOT pinned
	// to HookParam.BlockNumber (an L1 number there, see L2BlockByChain); it reads the latest state
	// together with the L2 block number, and rounds 2 and 3 pin to that block.
	var (
		usdg, navGuard, pm common.Address
		params             Params
		venue0, venue1     common.Hash
		l2Block            *big.Int
	)
	arbSys, isL2 := L2BlockByChain[chainID]
	req := param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides)
	if param.BlockNumber != nil && !isL2 {
		req.SetBlockNumber(param.BlockNumber)
	}
	req.
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "usdg"}, []any{&usdg}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "navGuard"}, []any{&navGuard}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "poolManager"}, []any{&pm}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "getParams"}, []any{&params}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "venueOf", Params: []any{tok0}}, []any{&venue0}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hookAddr, Method: "venueOf", Params: []any{tok1}}, []any{&venue1})
	if isL2 {
		req.AddCall(&ethrpc.Call{ABI: arbSysABI, Target: hexutil.Encode(arbSys[:]), Method: "arbBlockNumber"},
			[]any{&l2Block})
	}
	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}
	block := param.BlockNumber
	switch {
	case isL2:
		if !res.Result[6] || l2Block == nil {
			return nil, ErrPoolIsNotTracked
		}
		block = l2Block
	case block == nil:
		block = res.BlockNumber
	}

	var usdgIs0 bool
	var lot common.Address
	switch {
	case venue1 == poolID && (!res.Result[0] || usdg == tok0):
		usdgIs0, lot = true, tok1
	case venue0 == poolID && (!res.Result[0] || usdg == tok1):
		usdgIs0, lot = false, tok0
	default:
		return nil, ErrNotAVenue
	}
	usdgIdx := 1
	if usdgIs0 {
		usdgIdx = 0
	}
	usdgDecimals := p.Tokens[usdgIdx].Decimals

	key := quoteParams{}
	key.PoolKey.Currency0, key.PoolKey.Currency1, key.PoolKey.Hooks = tok0, tok1, param.HookAddress
	key.PoolKey.Fee = big.NewInt(int64(staticExtra.Fee))
	key.PoolKey.TickSpacing = big.NewInt(int64(staticExtra.TickSpacing))
	key.HookData = []byte{}

	// Round 2: NAV, float and the buy ladder (USDG in).
	var nav, float *big.Int
	buyIn := make([]*big.Int, len(rungsUsd))
	for i, usd := range rungsUsd {
		buyIn[i] = new(big.Int).Mul(new(big.Int).SetUint64(usd), bignumber.TenPowInt(usdgDecimals))
	}
	buyOut := make([]quoteResult, len(buyIn))
	req = param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides).SetBlockNumber(block)
	navCall := res.Result[1] && navGuard != (common.Address{})
	if navCall {
		req.AddCall(&ethrpc.Call{ABI: navGuardABI, Target: hexutil.Encode(navGuard[:]),
			Method: "checkedNavPerUnit18", Params: []any{lot}}, []any{&nav})
	}
	floatCall := res.Result[2] && pm != (common.Address{})
	if floatCall {
		req.AddCall(&ethrpc.Call{ABI: erc20ABI, Target: hexutil.Encode(lot[:]),
			Method: "balanceOf", Params: []any{pm}}, []any{&float})
	}
	addLadder(req, quoter, key, usdgIs0, buyIn, buyOut)
	res2, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	off := 0
	if navCall {
		if !res2.Result[off] {
			nav = nil
		}
		off++
	}
	if floatCall {
		if !res2.Result[off] {
			float = nil
		}
		off++
	}
	buy := toLadder(buyIn, buyOut, res2.Result[off:])

	// Round 3: the sell ladder (LOT in), sized at NAV (or, without one, at the first buy rung's
	// price), plus a rung just under the float.
	usdPerLot18 := nav
	if usdPerLot18 == nil && len(buy) > 0 {
		// USD18 per LOT from the first buy rung: in(usdg)*1e(18-dec)*1e18/out(lot)
		usdPerLot18 = new(big.Int).Mul(buy[0].In.ToBig(), bignumber.TenPowInt(36-usdgDecimals))
		usdPerLot18.Div(usdPerLot18, buy[0].Out.ToBig())
	}
	if usdPerLot18 == nil || usdPerLot18.Sign() == 0 {
		return nil, ErrEmptyLadder
	}
	sellIn := make([]*big.Int, 0, len(rungsUsd)+1)
	for _, usd := range rungsUsd {
		amt := new(big.Int).Mul(new(big.Int).SetUint64(usd), bignumber.TenPowInt(36))
		amt.Div(amt, usdPerLot18)
		if float != nil && amt.Cmp(float) >= 0 {
			break
		}
		sellIn = append(sellIn, amt)
	}
	if float != nil {
		for _, fb := range floatRungsBps {
			fr := new(big.Int).Mul(float, big.NewInt(fb))
			fr.Div(fr, big.NewInt(bps))
			if fr.Sign() > 0 && (len(sellIn) == 0 || fr.Cmp(sellIn[len(sellIn)-1]) > 0) {
				sellIn = append(sellIn, fr)
			}
		}
	}
	sellOut := make([]quoteResult, len(sellIn))
	req = param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides).SetBlockNumber(block)
	addLadder(req, quoter, key, !usdgIs0, sellIn, sellOut)
	res3, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	sell := toLadder(sellIn, sellOut, res3.Result)

	h.Extra = Extra{
		Tracked: true,
		UsdgIs0: usdgIs0,
		Nav:     u256OrNil(nav),
		Float:   u256OrNil(float),
		Buy:     buy,
		Sell:    sell,
		Block:   block.Uint64(),
	}
	if res.Result[3] {
		h.Extra.Params = &params
	}
	return json.Marshal(h)
}

func addLadder(req *ethrpc.Request, quoter common.Address, key quoteParams, zeroForOne bool,
	amounts []*big.Int, out []quoteResult) {
	for i, amt := range amounts {
		qp := key
		qp.ZeroForOne = zeroForOne
		qp.ExactAmount = amt
		req.AddCall(&ethrpc.Call{ABI: quoterABI, Target: hexutil.Encode(quoter[:]),
			Method: "quoteExactInputSingle", Params: []any{qp}}, []any{&out[i]})
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

func u256OrNil(x *big.Int) *uint256.Int {
	if x == nil {
		return nil
	}
	return uint256.MustFromBig(x)
}

// BeforeSwap takes the whole specified amount and pays the interpolated ladder output, so the
// pool's own CL math runs on zero. CalcOut (exact-in): in -= specified, out -= unspecified, hence
// DeltaSpecified = amountIn and DeltaUnspecified = -amountOut. CalcIn (the reverse of exact-in):
// out += specified, in += unspecified, hence DeltaSpecified = -amountOut, DeltaUnspecified = amountIn.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.Tracked {
		return nil, ErrPoolIsNotTracked
	}
	buy := params.ZeroForOne == h.UsdgIs0
	ladder := h.Sell
	if buy {
		ladder = h.Buy
	}
	if len(ladder) == 0 {
		return nil, ErrEmptyLadder
	}
	amt, overflow := uint256.FromBig(params.AmountSpecified)
	if overflow || params.AmountSpecified.Sign() <= 0 {
		return nil, ErrZeroOutput
	}

	if params.CalcOut {
		if !buy && h.Float != nil && amt.Gt(h.Float) {
			return nil, ErrSellExceedsFloat
		}
		out, gas, err := quoteOut(ladder, amt)
		if err != nil {
			return nil, err
		}
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   new(big.Int).Set(params.AmountSpecified),
			DeltaUnspecified: new(big.Int).Neg(out.ToBig()),
			Gas:              hookGas(gas),
		}, nil
	}

	in, gas, err := quoteIn(ladder, amt)
	if err != nil {
		return nil, err
	}
	if !buy && h.Float != nil && in.Gt(h.Float) {
		return nil, ErrSellExceedsFloat
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   new(big.Int).Neg(params.AmountSpecified),
		DeltaUnspecified: in.ToBig(),
		Gas:              hookGas(gas),
	}, nil
}

// hookGas: the quoter's estimate covers the whole swap; the v4 simulator adds its own base gas.
func hookGas(quoterGas int64) int64 {
	return max(quoterGas-uniswapv4.DefaultGas.BaseGas, 0)
}

// quoteOut interpolates the exact-in ladder at amountIn, rounding down.
func quoteOut(ladder []Rung, amountIn *uint256.Int) (*uint256.Int, int64, error) {
	top := ladder[len(ladder)-1]
	if amountIn.Gt(top.In) {
		return nil, 0, ErrBeyondLadder
	}
	var out, num, den uint256.Int
	i := 0
	for i < len(ladder) && amountIn.Gt(ladder[i].In) {
		i++
	}
	hi := ladder[i]
	if i == 0 {
		num.Mul(amountIn, hi.Out)
		out.Div(&num, hi.In)
	} else {
		lo := ladder[i-1]
		num.Sub(amountIn, lo.In)
		num.Mul(&num, den.Sub(hi.Out, lo.Out))
		den.Sub(hi.In, lo.In)
		out.Div(&num, &den)
		out.Add(&out, lo.Out)
	}
	if out.IsZero() {
		return nil, 0, ErrZeroOutput
	}
	return &out, hi.Gas, nil
}

// quoteIn inverts the same chords at amountOut, rounding the input up.
func quoteIn(ladder []Rung, amountOut *uint256.Int) (*uint256.Int, int64, error) {
	top := ladder[len(ladder)-1]
	if amountOut.Gt(top.Out) {
		return nil, 0, ErrBeyondLadder
	}
	var in, num, den, outDelta uint256.Int
	i := 0
	for i < len(ladder) && amountOut.Gt(ladder[i].Out) {
		i++
	}
	hi := ladder[i]
	if i == 0 {
		num.Mul(amountOut, hi.In)
		divUp(&in, &num, hi.Out)
	} else {
		lo := ladder[i-1]
		outDelta.Sub(amountOut, lo.Out)
		num.Mul(&outDelta, den.Sub(hi.In, lo.In))
		divUp(&in, &num, den.Sub(hi.Out, lo.Out))
		in.Add(&in, lo.In)
	}
	if in.IsZero() {
		return nil, 0, ErrZeroOutput
	}
	return &in, hi.Gas, nil
}

func divUp(z, x, y *uint256.Int) {
	var rem uint256.Int
	z.DivMod(x, y, &rem)
	if !rem.IsZero() {
		z.AddUint64(z, 1)
	}
}
