package everlongflamm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// FLAMMSwapLib (src/core/flamm/FLAMMSwapLib.sol @ c104 80abd43): one exact-input swap between poolAsset and a
// loan asset. Core prices the context, sizes the ceiling (the sell's gate room, notional cap and Router funding,
// re-planned on the fill's own consumption), asks the hook for the fee (floor rejects, cap clips) and the fill
// in the numeraire, converts at the edge (floor a payout, ceil a claim), validates it against the user's limit,
// the ceiling, the buy's notional cap and the loan asset's band around the checked cross, then settles through
// the Router (router.go) and re-asserts the gate. Validated end to end against the deployed pool
// (testdata/core_e2e_*.jsonl.gz): previewSwap grids with every revert class and executed sequences with their
// events and post-states.

// swapMaxFundingPasses is FLAMMSwapLib.MAX_FUNDING_PASSES.
const swapMaxFundingPasses = 4

// swapPlan is FLAMMSwapLib.Plan (:24) plus the book the priced fill materialised, which an execution commits.
type swapPlan struct {
	Idx         uint8
	PoolAssetIn bool
	PriceWad    uint256.Int
	Scale       uint256.Int
	CrossWad    uint256.Int
	SpotWad     uint256.Int
	FeeWad      uint256.Int
	Ceiling     uint256.Int // native units of the output token
	UsedNative  uint256.Int
	NetNative   uint256.Int
	GrossNative uint256.Int
	Sctx        swapContext
	Fill        hookFillResult // the hook's answer, loan leg in the numeraire
	book        hookBook
	capEvals    uint64 // _maxInForGrossCap solves of every previewExactIn the plan ran (gas accounting)
	passes      uint64 // funding passes the sell's plan ran (gas accounting; zero for a buy)
}

// swapResult is what FLAMM.swap returns and its Swap event carries (FLAMMStore.sol:131).
type swapResult struct {
	PoolAssetIn  bool
	AmountInUsed uint256.Int
	AmountOut    uint256.Int
	FeeOut       uint256.Int // grossNative - netNative, native units of the output token
	FeeWad       uint256.Int
	SpotAfterWad uint256.Int
	PriceWad     uint256.Int // the plan's loan-leg cross the band was checked at (not part of the return)
	capEvals     uint64      // _maxInForGrossCap solves of the whole transaction (gas accounting, not part of the return)
	passes       uint64      // the plan's funding passes (gas accounting, not part of the return)
}

// swapToN18 is FLAMMSwapLib.toN18 (:41): mulDiv(native * scale, q, WAD), the product checked.
func swapToN18(native, scale, q *uint256.Int) (uint256.Int, error) {
	n, err := gateMul(native, scale)
	if err != nil {
		return n, err
	}
	return mmMulDivOZ(&n, q, uWad)
}

// swapFromN18Floor is FLAMMSwapLib.fromN18Floor (:45): mulDiv(v, WAD, q) / scale, a payout floored twice.
func swapFromN18Floor(v, scale, q *uint256.Int) (uint256.Int, error) {
	x, err := mmMulDivOZ(v, uWad, q)
	if err != nil {
		return x, err
	}
	if scale.IsZero() {
		return x, errPanicDivZero
	}
	return *x.Div(&x, scale), nil
}

// swapFromN18Ceil is FLAMMSwapLib.fromN18Ceil (:49): ceilDiv(mulDiv(v, WAD, q, Up), scale), a claim ceiled twice.
func swapFromN18Ceil(v, scale, q *uint256.Int) (uint256.Int, error) {
	x, err := mmMulDivOZUp(v, uWad, q)
	if err != nil {
		return x, err
	}
	return ozCeilDiv(&x, scale)
}

// ozCeilDiv is OpenZeppelin 4.8 Math.ceilDiv: a == 0 ? 0 : (a - 1) / b + 1 (Panic(0x12) when b == 0 and a != 0).
func ozCeilDiv(a, b *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if a.IsZero() {
		return z, nil
	}
	if b.IsZero() {
		return z, errPanicDivZero
	}
	z.SubUint64(a, 1)
	z.Div(&z, b)
	return *z.AddUint64(&z, 1), nil
}

// previewSwap is FLAMMSwapLib.preview (:54), FLAMM.previewSwap's body: loan asset 0, no slippage limit. The
// view's read guard never binds outside a flow.
func (s *flammState) previewSwap(poolAssetIn bool, amountIn *uint256.Int, now uint64) (*swapPlan, error) {
	if amountIn.IsZero() {
		return nil, ErrInvalidAmount
	}
	pool := s.Pool
	b, err := s.priced(&pool, now)
	if err != nil {
		return nil, err
	}
	p, err := s.swapPlan(&pool, &b, 0, poolAssetIn, amountIn, now)
	if err != nil {
		return nil, err
	}
	return p, s.swapValidate(&pool, p, uZero)
}

// executeSwap is FLAMMSwapLib.execute (:65) as one transaction: the post-state is returned and the receiver is
// never written. The recipient check (`to == address(0)`) is the caller's business and not modelled. The
// committed fee and fill (executeFeeWad, executeExactIn) are the same pure functions of the same storage and
// context the plan priced, so FeeMismatch and FillMismatch cannot fire and the plan's book is committed as is.
func (s *flammState) executeSwap(tokenIn, tokenOut common.Address, amountIn, minAmountOut *uint256.Int, deadline,
	now uint64) (*swapResult, *flammState, error) {
	if now > deadline {
		return nil, nil, ErrExpired
	}
	if amountIn.IsZero() {
		return nil, nil, ErrInvalidAmount
	}
	idx, poolAssetIn, err := s.pair(tokenIn, tokenOut)
	if err != nil {
		return nil, nil, err
	}
	post := s.clone()
	b, err := post.priced(&post.Pool, now)
	if err != nil {
		return nil, nil, err
	}
	p, err := post.swapPlan(&post.Pool, &b, idx, poolAssetIn, amountIn, now)
	if err != nil {
		return nil, nil, err
	}
	if err = post.swapValidate(&post.Pool, p, minAmountOut); err != nil {
		return nil, nil, err
	}
	hook, err := post.Hooks.Swap.port()
	if err != nil {
		return nil, nil, err
	}
	hook.commit(&p.book) // validate refused a zero amountInUsed: executeExactIn commits
	if poolAssetIn {
		err = mmSettleSellLegs(&post.Pool, &post.Router, idx, &p.PriceWad, &p.UsedNative, &p.NetNative, now)
	} else {
		err = mmSettleBuyLegs(&post.Pool, &post.Router, idx, &p.UsedNative, &p.NetNative, now)
	}
	if err != nil {
		return nil, nil, err
	}
	post.Router.endTransaction()
	post.Pool.PriceWad, post.Pool.CrossWad = nil, nil // a frame of this timestamp, not state
	// executeExactIn re-runs the plan's last fill on the same context (FLAMMSwapLib.sol:87), bisection included.
	r := &swapResult{PoolAssetIn: poolAssetIn, AmountInUsed: p.UsedNative, AmountOut: p.NetNative,
		FeeWad: p.FeeWad, SpotAfterWad: p.Fill.SpotAfterWad, PriceWad: p.PriceWad,
		capEvals: p.capEvals + p.Fill.capEvals, passes: p.passes}
	r.FeeOut.Sub(&p.GrossNative, &p.NetNative) // fromN18Floor is monotone and feeOut <= grossOut
	return r, post, nil
}

// swapPlan is FLAMMSwapLib._plan (:125) over the priced book b (pool carries its price frame).
func (s *flammState) swapPlan(pool *gatePool, b *gateBook, idx uint8, poolAssetIn bool, amountIn *uint256.Int,
	now uint64) (*swapPlan, error) {
	if int(idx) >= len(pool.Loans) || int(idx) >= len(b.Legs) {
		return nil, errPanicIndex
	}
	cfg := &pool.Loans[idx]
	if poolAssetIn {
		if s.Paused {
			return nil, ErrPaused
		}
		if err := s.feature(flammFeatureSwapSell); err != nil {
			return nil, err
		}
	} else if err := s.feature(flammFeatureSwapBuy); err != nil {
		return nil, err
	}
	p0, priceTs, err := s.price(now)
	if err != nil {
		return nil, err
	}
	if poolAssetIn {
		ok, err := s.pegOk(int(idx), now)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrPegBroken
		}
	}
	p := &swapPlan{Idx: idx, PoolAssetIn: poolAssetIn, Scale: cfg.Scale, PriceWad: b.Legs[idx].PriceWad,
		CrossWad: b.Legs[idx].CrossWad}
	if p.PriceWad.IsZero() || p.CrossWad.IsZero() {
		return nil, ErrPriceUnchecked
	}
	ctx, err := gateContext(b, &p0, priceTs, &s.ShareSupply)
	if err != nil {
		return nil, err
	}
	hook, err := s.Hooks.Swap.port()
	if err != nil {
		return nil, err
	}
	if p.SpotWad, err = hook.spot(); err != nil {
		return nil, err
	}
	if !poolAssetIn {
		if p.Ceiling, err = gateGross(b); err != nil {
			return nil, err
		}
		return p, s.swapFill(pool, p, &ctx, amountIn)
	}
	// Sell ceiling: the gate room, the notional cap and the Router's funding of the loan leg, which counts the
	// incoming poolAsset as collateral; a partial fill is re-planned on its own consumption.
	u, err := gateExposurePW(b)
	if err != nil {
		return nil, err
	}
	room, err := gateRoomNative(pool, b, int(idx), &u)
	if err != nil {
		return nil, err
	}
	if !cfg.MaxSwapNotional.IsZero() && cfg.MaxSwapNotional.Lt(&room) {
		room = cfg.MaxSwapNotional
	}
	liquid := &b.Legs[idx].Liquid
	collateralIn := *amountIn
	for pass := 0; pass < swapMaxFundingPasses; pass++ {
		p.passes++
		funding, err := s.swapFunding(b, idx, &collateralIn, &p.PriceWad, now)
		if err != nil {
			return nil, err
		}
		if funding, err = gateAdd(liquid, &funding); err != nil {
			return nil, err
		}
		p.Ceiling = *minU(&room, &funding) // `room < funding ? room : funding`: equal on a tie
		if p.Ceiling.IsZero() {
			return nil, ErrRoomExhausted
		}
		if err = s.swapFill(pool, p, &ctx, amountIn); err != nil {
			return nil, err
		}
		if !p.UsedNative.Lt(&collateralIn) {
			return p, nil
		}
		collateralIn = p.UsedNative
		var need uint256.Int
		if p.NetNative.Gt(liquid) {
			need.Sub(&p.NetNative, liquid)
		}
		funding, err = s.swapFunding(b, idx, &collateralIn, &p.PriceWad, now)
		if err != nil {
			return nil, err
		}
		if !need.Gt(&funding) {
			return p, nil
		}
	}
	return nil, ErrOutputAboveCeiling
}

// swapFunding is `router.fundingCeiling(pool, idx, b.physical + collateralIn, priceWad)`, the sum checked.
func (s *flammState) swapFunding(b *gateBook, idx uint8, collateralIn, priceWad *uint256.Int, now uint64) (uint256.Int,
	error) {
	coll, err := gateAdd(&b.Physical, collateralIn)
	if err != nil {
		return coll, err
	}
	return s.Router.fundingCeiling(idx, &coll, priceWad, now)
}

// swapFill is FLAMMSwapLib._fill (:176): the swap hook's fee, floored by the larger of the pool's and the asset's
// floor (a rejection) and capped by the pool's cap (a clip), then the swap hook's fill on the clipped fee converted
// to native units -- a sell's payout floored, a buy's claim ceiled (and never above amountIn). The fee and
// invariant roles are one hook (hook_registry.go resolveHooksIn).
func (s *flammState) swapFill(pool *gatePool, p *swapPlan, ctx *poolContext, amountIn *uint256.Int) error {
	hook, err := s.Hooks.Swap.port()
	if err != nil {
		return err
	}
	cfg := &pool.Loans[p.Idx]
	p.Sctx = swapContext{Pool: *ctx, PoolAssetIn: p.PoolAssetIn, LoanIndex: p.Idx, CrossWad: p.CrossWad,
		FeeFloorWad: cfg.FeeFloorWad}
	if p.PoolAssetIn {
		p.Sctx.AmountIn = *amountIn
		p.Sctx.MaxAmountOut, err = swapToN18(&p.Ceiling, &p.Scale, &p.CrossWad)
	} else {
		p.Sctx.AmountIn, err = swapToN18(amountIn, &p.Scale, &p.CrossWad)
		p.Sctx.MaxAmountOut = p.Ceiling
	}
	if err != nil {
		return err
	}
	if p.FeeWad, err = hook.previewFeeWad(&p.Sctx); err != nil {
		return err
	}
	floorWad := &s.FeeFloorWad
	if p.Sctx.FeeFloorWad.Gt(floorWad) {
		floorWad = &p.Sctx.FeeFloorWad
	}
	if p.FeeWad.Lt(floorWad) {
		return ErrFeeOutOfBounds
	}
	if p.FeeWad.Gt(&s.FeeCapWad) {
		p.FeeWad = s.FeeCapWad
	}
	if p.Fill, p.book, err = hook.fill(&p.Sctx, &p.FeeWad); err != nil {
		return err
	}
	p.capEvals += p.Fill.capEvals
	f := &p.Fill
	if p.PoolAssetIn {
		p.UsedNative = f.AmountInUsed
		if p.GrossNative, err = swapFromN18Floor(&f.GrossOut, &p.Scale, &p.CrossWad); err != nil {
			return err
		}
		p.NetNative.Clear()
		if !f.FeeOut.Gt(&f.GrossOut) {
			var net uint256.Int
			net.Sub(&f.GrossOut, &f.FeeOut)
			if p.NetNative, err = swapFromN18Floor(&net, &p.Scale, &p.CrossWad); err != nil {
				return err
			}
		}
		return nil
	}
	p.UsedNative.Clear()
	if !f.AmountInUsed.IsZero() {
		if p.UsedNative, err = swapFromN18Ceil(&f.AmountInUsed, &p.Scale, &p.CrossWad); err != nil {
			return err
		}
	}
	if p.UsedNative.Gt(amountIn) {
		return ErrFillInvalid
	}
	p.GrossNative = f.GrossOut
	p.NetNative.Clear()
	if !f.FeeOut.Gt(&f.GrossOut) {
		p.NetNative.Sub(&f.GrossOut, &f.FeeOut)
	}
	return nil
}

// swapValidate is FLAMMSwapLib._validate (:211): a well-formed non-zero fill, the user's limit, the ceiling, the
// buy's notional cap, and the NET inside the loan asset's band around the checked cross (both legs valued in
// L18: a sell's net*scale against used*priceWad, a buy's net*priceWad against used*scale).
func (s *flammState) swapValidate(pool *gatePool, p *swapPlan, minAmountOut *uint256.Int) error {
	f := &p.Fill
	if f.AmountInUsed.IsZero() || f.AmountInUsed.Gt(&p.Sctx.AmountIn) || f.FeeOut.Gt(&f.GrossOut) {
		return ErrFillInvalid
	}
	net := &p.NetNative
	if p.UsedNative.IsZero() || net.IsZero() {
		return ErrFillInvalid
	}
	if net.Lt(minAmountOut) {
		return ErrSlippage
	}
	if net.Gt(&p.Ceiling) {
		return ErrOutputAboveCeiling
	}
	cfg := &pool.Loans[p.Idx]
	var outValue, inValue uint256.Int
	var err error
	if p.PoolAssetIn {
		if outValue, err = gateMul(net, &p.Scale); err != nil {
			return err
		}
		if inValue, err = gateMul(&p.UsedNative, &p.PriceWad); err != nil {
			return err
		}
	} else {
		if !cfg.MaxSwapNotional.IsZero() && p.UsedNative.Gt(&cfg.MaxSwapNotional) {
			return ErrNotionalCap
		}
		if outValue, err = gateMul(net, &p.PriceWad); err != nil {
			return err
		}
		if inValue, err = gateMul(&p.UsedNative, &p.Scale); err != nil {
			return err
		}
	}
	return swapBand(&outValue, &inValue, &cfg.SwapPriceBandWad)
}

// swapBand is the shared band test `out < mulDiv(in, WAD - band, WAD) || out > mulDiv(in, WAD + band, WAD)`,
// evaluated left to right.
func swapBand(outValue, inValue, bandWad *uint256.Int) error {
	if bandWad.Gt(uWad) {
		return errPanicArithmetic
	}
	var k uint256.Int
	lo, err := mmMulDivOZ(inValue, k.Sub(uWad, bandWad), uWad)
	if err != nil {
		return err
	}
	if outValue.Lt(&lo) {
		return ErrPriceBand
	}
	hi, err := mmMulDivOZ(inValue, k.Add(uWad, bandWad), uWad)
	if err != nil {
		return err
	}
	if outValue.Gt(&hi) {
		return ErrPriceBand
	}
	return nil
}
