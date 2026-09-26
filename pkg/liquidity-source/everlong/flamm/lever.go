package everlongflamm

import (
	"github.com/holiman/uint256"
)

// FLAMMLeverLib (src/core/flamm/FLAMMLeverLib.sol @ c104 80abd43): the leverage venue. Core builds the frame and
// the credit-zero room, resolves the spread through a low-level staticcall to the spread hook, asks the stateless
// leverage hook (levhook.go) for the fill, and checks it against the checked feed -- value leak, the taker band,
// the room, the CR floor and the Router's funding for a lever-up; the ceiling, the concession and the band for a
// lever-down -- before settling through the swap route's Router bundles (router.go). The swap hook's fee and
// invariant are never called and its stored book never moves: the next swap's lazy rescale absorbs the fill.

var (
	leverCrFloorWad       = uint256.NewInt(1_820_000_000_000_000_000) // PHYSICAL_CR_FLOOR_WAD = 1.82e18
	leverSpreadFloorPpm   = uint256.NewInt(2_500)                     // LEV_SPREAD_FLOOR_PPM
	leverSpreadCeilingPpm = uint256.NewInt(100_000)                   // LEV_SPREAD_CEILING_PPM
	leverBandToPpm        = uint256.NewInt(1_000_000_000_000)         // swapPriceBandWad / 1e12 is the band in ppm
)

// leverPlan is FLAMMLeverLib.Plan (:27).
type leverPlan struct {
	PriceWad   uint256.Int
	Ctx        leverContext
	Fill       *levFill
	PayL18     uint256.Int // the taker's real loan leg, L18: paid to them (up) or by them (down)
	Out        uint256.Int // up: loan asset native; down: poolAsset native
	PayNative  uint256.Int // down only: the loan asset pulled from the taker
	SpreadLive bool        // the hook answered; a degraded fill never becomes the next degrade value
}

// leverResult is what leverUp / leverDown return and their LeverUp / LeverDown events carry (FLAMMStore.sol:141).
type leverResult struct {
	Up           bool
	AmountInUsed uint256.Int // up: poolAsset used; down: payNative
	AmountOut    uint256.Int
	SpreadPpm    uint256.Int
	CrAfterWad   uint256.Int
	PayL18       uint256.Int // the taker's loan leg the band was checked on (not part of the return)
	PriceWad     uint256.Int // the checked cross (not part of the return)
}

// previewLever is FLAMMLeverLib.preview (:37), FLAMM.previewLever's body. AmountInUsed is the hook's used input
// up and payNative down.
func (s *flammState) previewLever(up bool, amountIn *uint256.Int, now uint64) (*leverResult, error) {
	if amountIn.IsZero() {
		return nil, ErrInvalidAmount
	}
	pool := s.Pool
	p, err := s.leverPlanFor(&pool, up, amountIn, now)
	if err != nil {
		return nil, err
	}
	return p.result(up), nil
}

func (p *leverPlan) result(up bool) *leverResult {
	r := &leverResult{Up: up, AmountOut: p.Out, SpreadPpm: p.Ctx.SpreadPpm, CrAfterWad: *p.Fill.CrAfterWad,
		PayL18: p.PayL18, PriceWad: p.PriceWad}
	if up {
		r.AmountInUsed = *p.Fill.AmountInUsed
	} else {
		r.AmountInUsed = p.PayNative
	}
	return r
}

func (s *flammState) leverPlanFor(pool *gatePool, up bool, amountIn *uint256.Int, now uint64) (*leverPlan, error) {
	if up {
		return s.leverPlanUp(pool, amountIn, now)
	}
	return s.leverPlanDown(pool, amountIn, now)
}

// executeLever is FLAMMLeverLib.leverUp (:47) / leverDown (:62) as one transaction: the post-state is returned
// and the receiver is never written. A live spread is stored as the next degrade value before the fill;
// executeLever is previewLever on the same context (the hook is stateless), so FillMismatch cannot fire. A
// lever-up settles as a sell of the used poolAsset for `out`; a lever-down anchors the entry gate, settles as a
// buy of `out` poolAsset for payNative and re-asserts it.
func (s *flammState) executeLever(up bool, amountIn, minOut *uint256.Int, deadline, now uint64) (*leverResult,
	*flammState, error) {
	if now > deadline {
		return nil, nil, ErrExpired
	}
	if amountIn.IsZero() {
		return nil, nil, ErrInvalidAmount
	}
	post := s.clone()
	p, err := post.leverPlanFor(&post.Pool, up, amountIn, now)
	if err != nil {
		return nil, nil, err
	}
	if p.Out.Lt(minOut) {
		return nil, nil, ErrSlippage
	}
	if p.SpreadLive {
		post.LastLeverSpreadPpm.SetUint64(p.Ctx.SpreadPpm.Uint64() & 0xffffffff) // uint32(spreadPpm)
	}
	if up {
		err = mmSettleSellLegs(&post.Pool, &post.Router, 0, &p.PriceWad, p.Fill.AmountInUsed, &p.Out, now)
	} else {
		err = mmSettleBuyLegs(&post.Pool, &post.Router, 0, &p.PayNative, &p.Out, now)
	}
	if err != nil {
		return nil, nil, err
	}
	post.Router.endTransaction()
	post.Pool.PriceWad, post.Pool.CrossWad = nil, nil // a frame of this timestamp, not state
	return p.result(up), post, nil
}

// leverOpen is FLAMMLeverLib._open (:80).
func (s *flammState) leverOpen(up bool, now uint64) error {
	if !s.Hooks.hasLeverage() {
		return ErrLeverageDisabled
	}
	if s.LevPaused {
		return ErrLevPaused
	}
	if err := s.feature(flammFeatureLeverage); err != nil {
		return err
	}
	if up {
		if s.Paused {
			return ErrPaused
		}
		ok, err := s.pegOk(0, now)
		if err != nil {
			return err
		}
		if !ok {
			return ErrPegBroken
		}
	}
	return nil
}

// leverFrame is the shared head of _planUp / _planDown: the checked cross, the priced book and its context.
func (s *flammState) leverFrame(pool *gatePool, p *leverPlan, now uint64) (gateBook, poolContext, error) {
	priceWad, priceTs, err := s.price(now)
	if err != nil {
		return gateBook{}, poolContext{}, err
	}
	p.PriceWad = priceWad
	b, err := s.priced(pool, now)
	if err != nil {
		return b, poolContext{}, err
	}
	if len(b.Legs) == 0 {
		return b, poolContext{}, errPanicIndex
	}
	ctx, err := gateContext(&b, &priceWad, priceTs, &s.ShareSupply)
	return b, ctx, err
}

// leverFill is `ILeverageInvariantHook(leverageHook).previewLever(p.ctx)` (:100, :128): the leverage hook's quote
// over the pool's swap hook.
func (s *flammState) leverFill(ctx *leverContext) (*levFill, error) {
	lev, err := s.Hooks.Leverage.port()
	if err != nil {
		return nil, err
	}
	swap, err := s.Hooks.Swap.port()
	if err != nil {
		return nil, err
	}
	return lev.previewLever(ctx, swap)
}

// leverPlanUp is FLAMMLeverLib._planUp (:90).
func (s *flammState) leverPlanUp(pool *gatePool, volIn *uint256.Int, now uint64) (*leverPlan, error) {
	if err := s.leverOpen(true, now); err != nil {
		return nil, err
	}
	p := &leverPlan{}
	b, ctx, err := s.leverFrame(pool, p, now)
	if err != nil {
		return nil, err
	}
	leg := &b.Legs[0]
	head, err := s.leverRoom(pool, &b)
	if err != nil {
		return nil, err
	}
	p.Ctx = leverContext{Pool: ctx, Up: true, AmountIn: *volIn, MaxOut: head}
	if p.Ctx.SpreadPpm, p.SpreadLive, err = s.leverSpread(pool, true, now); err != nil {
		return nil, err
	}
	f, err := s.leverFill(&p.Ctx)
	if err != nil {
		return nil, err
	}
	if f.AmountInUsed.IsZero() || f.AmountInUsed.Gt(volIn) || !f.GrossOut.Gt(f.VirtualLegL18) {
		return nil, ErrFillInvalid
	}
	p.Fill = f
	if leg.Scale.IsZero() {
		return nil, errPanicDivZero
	}
	p.Out.Sub(f.GrossOut, f.VirtualLegL18).Div(&p.Out, &leg.Scale)
	if p.Out.IsZero() {
		return nil, ErrFillInvalid
	}
	p.PayL18.Mul(&p.Out, &leg.Scale) // <= grossOut - virtualLeg
	value, err := gateMul(f.AmountInUsed, &p.PriceWad)
	if err != nil {
		return nil, err
	}
	// The book's NAV change at the feed is `volIn * price - paid`: never less than the spread.
	var keep uint256.Int
	leak, err := mmMulDivOZ(&value, keep.Sub(uPpm, &p.Ctx.SpreadPpm), uPpm) // spreadPpm < PPM
	if err != nil {
		return nil, err
	}
	if p.PayL18.Gt(&leak) {
		return nil, ErrLevValueLeak
	}
	if err = leverBandFloor(&p.PayL18, &value, &pool.Loans[0].SwapPriceBandWad); err != nil {
		return nil, err
	}
	// Credit zero, all or nothing: the incoming poolAsset earns no room.
	if p.PayL18.Gt(&head) {
		return nil, ErrRoomExceeded
	}
	if f.CrAfterWad.Lt(leverCrFloorWad) {
		return nil, ErrLevBelowFloor
	}
	var rest uint256.Int
	if p.Out.Gt(&leg.Liquid) {
		rest.Sub(&p.Out, &leg.Liquid)
	}
	coll, err := gateAdd(&ctx.PhysicalPoolAsset, f.AmountInUsed)
	if err != nil {
		return nil, err
	}
	funding, err := s.Router.fundingCeiling(0, &coll, &p.PriceWad, now)
	if err != nil {
		return nil, err
	}
	if rest.Gt(&funding) {
		return nil, ErrOutputAboveCeiling
	}
	return p, nil
}

// leverPlanDown is FLAMMLeverLib._planDown (:119).
func (s *flammState) leverPlanDown(pool *gatePool, loanIn *uint256.Int, now uint64) (*leverPlan, error) {
	if err := s.leverOpen(false, now); err != nil {
		return nil, err
	}
	p := &leverPlan{}
	b, ctx, err := s.leverFrame(pool, p, now)
	if err != nil {
		return nil, err
	}
	leg := &b.Legs[0]
	inL18, err := gateMul(loanIn, &leg.Scale)
	if err != nil {
		return nil, err
	}
	maxOut, err := gateAdd(&ctx.PhysicalPoolAsset, &ctx.PostedPoolAsset)
	if err != nil {
		return nil, err
	}
	p.Ctx = leverContext{Pool: ctx, AmountIn: inL18, MaxOut: maxOut}
	if p.Ctx.SpreadPpm, p.SpreadLive, err = s.leverSpread(pool, false, now); err != nil {
		return nil, err
	}
	f, err := s.leverFill(&p.Ctx)
	if err != nil {
		return nil, err
	}
	if f.AmountInUsed.IsZero() || f.AmountInUsed.Gt(&inL18) || !f.AmountInUsed.Gt(f.VirtualLegL18) || f.GrossOut.IsZero() {
		return nil, ErrFillInvalid
	}
	if f.GrossOut.Gt(&maxOut) {
		return nil, ErrOutputAboveCeiling
	}
	p.Fill = f
	p.Out = *f.GrossOut
	p.PayL18.Sub(f.AmountInUsed, f.VirtualLegL18)
	outValue, err := gateMul(f.GrossOut, &p.PriceWad)
	if err != nil {
		return nil, err
	}
	// The venue never releases more than the concession beyond the value it takes in, at the feed.
	concession, err := mmMulDivOZ(&p.PayL18, levMaxConcessionPpm, uPpm)
	if err != nil {
		return nil, err
	}
	if outValue.Gt(&concession) {
		return nil, ErrLevValueLeak
	}
	if err = leverBandFloor(&outValue, &p.PayL18, &pool.Loans[0].SwapPriceBandWad); err != nil {
		return nil, err
	}
	if p.PayNative, err = ozCeilDiv(&p.PayL18, &leg.Scale); err != nil {
		return nil, err
	}
	if p.PayNative.Gt(loanIn) {
		p.PayNative = *loanIn
	}
	return p, nil
}

// leverBandFloor is the one-sided taker band `got < mulDiv(worth, WAD - band, WAD)` reverting PriceBand.
func leverBandFloor(got, worth, bandWad *uint256.Int) error {
	if bandWad.Gt(uWad) {
		return errPanicArithmetic
	}
	var k uint256.Int
	floor, err := mmMulDivOZ(worth, k.Sub(uWad, bandWad), uWad)
	if err != nil {
		return err
	}
	if got.Lt(&floor) {
		return ErrPriceBand
	}
	return nil
}

// leverRoom is FLAMMLeverLib._room (:147): loan asset 0's credit-zero headroom (L18) less the epsilon shave;
// no phi lift, since the incoming poolAsset is not credited.
func (s *flammState) leverRoom(pool *gatePool, b *gateBook) (uint256.Int, error) {
	u, err := gateExposurePW(b)
	if err != nil {
		return u, err
	}
	head, err := gateHeadOf(b, 0, &u, &pool.LtvWad)
	if err != nil {
		return head, err
	}
	shave, err := mmMulDivOZ(&head, &pool.RoomEpsilonWad, uWad)
	if err != nil {
		return head, err
	}
	if shave.Gt(&head) {
		return head, errPanicArithmetic
	}
	return *head.Sub(&head, &shave), nil
}

// leverSpread is FLAMMLeverLib._spread (:158). No answer -- no spread hook, a stale post, or ppm >= PPM -- fails
// a lever-up closed and degrades a lever-down to the last live spread (the venue ceiling before any live fill),
// unclamped. A live answer is clamped into [LEV_SPREAD_FLOOR_PPM, min(swapPriceBandWad / 1e12, 100_000)].
func (s *flammState) leverSpread(pool *gatePool, up bool, now uint64) (uint256.Int, bool, error) {
	ok, sp, err := s.Hooks.Spread.spreadPpm(now)
	if err != nil {
		return sp, false, err
	}
	if !ok || !sp.Lt(uPpm) {
		if up {
			return sp, false, ErrSpreadUnavailable
		}
		if s.LastLeverSpreadPpm.IsZero() {
			return *leverSpreadCeilingPpm, false, nil
		}
		return s.LastLeverSpreadPpm, false, nil
	}
	if sp.Lt(leverSpreadFloorPpm) {
		sp = *leverSpreadFloorPpm
	}
	var ceiling uint256.Int
	ceiling.Div(&pool.Loans[0].SwapPriceBandWad, leverBandToPpm)
	if ceiling.Gt(leverSpreadCeilingPpm) {
		ceiling = *leverSpreadCeilingPpm
	}
	if sp.Gt(&ceiling) {
		sp = ceiling
	}
	return sp, true, nil
}
