package rateob

import (
	"github.com/holiman/uint256"
)

// poolLegResult is the band pool's part of a quote.
type poolLegResult struct {
	in, out, fee uint256.Int // input the pool takes, net output it pays, the fee it keeps
	limitPrice   uint256.Int // the order price the executor sends
	lmp          uint256.Int // lmp after the swap (the walk's, then the pool's report)
	fills        []bandFill
}

// poolLeg ports what the engine does with a taker remainder once the book walk is over:
// OrderPlacementLib.detMake -> PoolFallbackLib.route -> BandPool.swap.
//
//   - The bound. route asks the pool for at least _poolBound(convert(makePrice, remaining,
//     !isBid)): the remainder valued at the order's make price, less band 0's fee. The make
//     price is MarketMakePriceLib.limitBuy/limitSell of the lmp the walk left, the order price
//     and the heads after the walk, so it depends on the price the executor sends. The
//     simulator picks that price (see orderPrice) and checks the bound exactly.
//   - All or nothing. The pool reverts below the bound (caught: the whole remainder is
//     refunded), and whatever it does not take is refunded. So the remainder sent to the pool
//     must be one the pool takes in full and pays at least the bound for: the leg is the largest
//     such amount, tried at the full remainder and then at each band boundary, and the rest of
//     the input is returned as RemainingTokenAmountIn rather than sent.
//
// It returns nil when no amount passes, and the caller quotes book-only.
func (p *PoolSimulator) poolLeg(r *sweepResult, amountIn *uint256.Int, isBuy bool) (*poolLegResult, error) {
	limitPrice, makePrice, ok := p.orderPrice(r, isBuy)
	if !ok {
		return nil, nil
	}
	var bookUsed uint256.Int
	bookUsed.Sub(amountIn, &r.remaining)

	full, err := p.bandWalk(&r.remaining, isBuy)
	if err != nil || full.out.IsZero() {
		return nil, nil //nolint:nilerr // a pool that would revert is skipped, as route's try/catch does
	}
	// Candidates, largest first: what the pool takes of the whole remainder, then the input
	// that exactly empties each band it reached, in reverse.
	candidates := make([]uint256.Int, 0, len(full.fills)+1)
	candidates = append(candidates, full.in)
	var prefix uint256.Int
	prefixes := make([]uint256.Int, len(full.fills))
	for i := range full.fills {
		prefix.Add(&prefix, &full.fills[i].in)
		prefixes[i].Set(&prefix)
	}
	for i := len(prefixes) - 1; i >= 0; i-- {
		if !prefixes[i].Eq(&candidates[len(candidates)-1]) && !prefixes[i].IsZero() {
			candidates = append(candidates, prefixes[i])
		}
	}

	rate0 := p.bandRate(0)
	var gross, minOut, total uint256.Int
	for i := range candidates {
		in := &candidates[i]
		w, err := p.bandWalk(in, isBuy)
		// BandPool.swap reverts NoLiquidity on zero output, and the walk must take all of `in`
		// or the engine refunds the rest.
		if err != nil || w.out.IsZero() || !w.in.Eq(in) {
			continue
		}
		// PoolFallbackLib._poolBound(convert(makePrice, remaining, !isBid)); a zero bound is not
		// routed at all.
		if convert(&gross, makePrice, in, !isBuy, p.DecDiff, p.BaseBQuote) != nil || gross.IsZero() {
			continue
		}
		if _, overflow := minOut.MulOverflow(&gross, rate0); overflow {
			continue
		}
		minOut.Sub(&gross, minOut.Div(&minOut, denom))
		if minOut.IsZero() || w.out.Lt(&minOut) {
			continue
		}
		// MatchingEngine._deposit sizes the whole order at the price sent.
		total.Add(&bookUsed, in)
		if ok, err := p.sizeOK(limitPrice, &total, isBuy); err != nil || !ok {
			return nil, nil //nolint:nilerr // an order the engine would refuse is quoted book-only
		}
		leg := &poolLegResult{fills: w.fills}
		leg.in.Set(in)
		leg.out.Set(&w.out)
		leg.fee.Set(&w.fee)
		leg.limitPrice.Set(limitPrice)
		p.report(&leg.lmp, &r.walkLmp, &w.matched)
		return leg, nil
	}
	return nil, nil
}

// orderPrice picks the price the executor sends with a pool leg, and returns it with the make
// price the engine derives from it after the walk.
//
// The book walk must not change: limitBuy matches up to min(price, ceil(lmp*(D+s)/D)) and
// limitSell down to max(price, floor(lmp*(D-s)/D)), so a buy price at or above that reach and a
// sell price at or below it match exactly what the walk did. Within that, the price is the one
// that makes the pool's bound loosest -- the highest make price for a buy (fewer base asked for
// the remainder), the lowest for a sell -- because the bound is the only way it affects the fill:
//
//	buy:  price = max(reach, ceil(walkLmp*(D+s)/D)), make = min(price, ceil(walkLmp*(D+s)/D), askHead)
//	sell: price = min(reach, floor(walkLmp*(D-s)/D)), make = max(price, floor(walkLmp*(D-s)/D), ...)
//
// walkLmp is the lmp the walk leaves (the last level it entered), not the snapshot's.
func (p *PoolSimulator) orderPrice(r *sweepResult, isBuy bool) (limitPrice, makePrice *uint256.Int, ok bool) {
	var price, mk uint256.Int
	if isBuy {
		up, ok := capUp(&r.walkLmp, p.SpreadBuy)
		if !ok {
			return nil, nil, false
		}
		price.Set(&r.bound)
		if up.Gt(&price) {
			price.Set(&up)
		}
		var bidHead uint256.Int
		if len(p.Bids) > 0 {
			bidHead.Set(&p.Bids[0].Price)
		}
		if !mmLimitBuy(&mk, &r.walkLmp, &price, &bidHead, &r.walkHead, p.SpreadBuy) {
			return nil, nil, false
		}
	} else {
		down, ok := capDown(&r.walkLmp, p.SpreadSell)
		if !ok {
			return nil, nil, false
		}
		price.Set(&r.bound)
		if down.Lt(&price) {
			price.Set(&down)
		}
		var askHead uint256.Int
		if len(p.Asks) > 0 {
			askHead.Set(&p.Asks[0].Price)
		}
		if !mmLimitSell(&mk, &r.walkLmp, &price, &r.walkHead, &askHead, p.SpreadSell) {
			return nil, nil, false
		}
	}
	if price.IsZero() || mk.IsZero() {
		return nil, nil, false
	}
	return &price, &mk, true
}

// capUp is MarketMakePriceLib._up: ceil(x*(D+s)/D).
func capUp(x *uint256.Int, s uint32) (uint256.Int, bool) {
	var z, t uint256.Int
	if _, overflow := z.MulOverflow(x, t.AddUint64(denom, uint64(s))); overflow {
		return z, false
	}
	z.Add(&z, t.SubUint64(denom, 1)).Div(&z, denom)
	return z, true
}

// capDown is floor(x*(D-s)/D); s < D is checked by NewPoolSimulator.
func capDown(x *uint256.Int, s uint32) (uint256.Int, bool) {
	var z, t uint256.Int
	if _, overflow := z.MulOverflow(x, t.SubUint64(denom, uint64(s))); overflow {
		return z, false
	}
	z.Div(&z, denom)
	return z, true
}

// mmLimitBuy is MarketMakePriceLib.limitBuy(lmp, lp, bidHead, askHead, spread).
func mmLimitBuy(z, lmp, lp, bidHead, askHead *uint256.Int, spread uint32) bool {
	minLp := func(up *uint256.Int) {
		if lp.Lt(up) {
			z.Set(lp)
		} else {
			z.Set(up)
		}
	}
	switch {
	case askHead.IsZero() && bidHead.IsZero():
		if lmp.IsZero() {
			z.Set(lp)
			return true
		}
		up, ok := capUp(lmp, spread)
		if !ok {
			return false
		}
		minLp(&up)
	case askHead.IsZero():
		src := lmp
		if lmp.IsZero() {
			src = bidHead
		}
		up, ok := capUp(src, spread)
		if !ok {
			return false
		}
		minLp(&up)
	case bidHead.IsZero() || !lmp.IsZero():
		src := lmp
		if lmp.IsZero() {
			src = askHead
		}
		up, ok := capUp(src, spread)
		if !ok {
			return false
		}
		minLp(&up)
		if !z.Lt(askHead) {
			z.Set(askHead)
		}
	default: // both heads, no lmp
		if lp.Lt(askHead) {
			z.Set(lp)
		} else {
			z.Set(askHead)
		}
	}
	return true
}

// mmLimitSell is MarketMakePriceLib.limitSell(lmp, lp, bidHead, askHead, spread).
func mmLimitSell(z, lmp, lp, bidHead, askHead *uint256.Int, spread uint32) bool {
	maxLp := func(down *uint256.Int) {
		if lp.Gt(down) {
			z.Set(lp)
		} else {
			z.Set(down)
		}
	}
	switch {
	case askHead.IsZero() && bidHead.IsZero():
		if lmp.IsZero() {
			z.Set(lp)
			return true
		}
		down, ok := capDown(lmp, spread)
		if !ok {
			return false
		}
		maxLp(&down)
	case askHead.IsZero():
		src := lmp
		if lmp.IsZero() {
			src = bidHead
		}
		down, ok := capDown(src, spread)
		if !ok {
			return false
		}
		maxLp(&down)
		if !z.Gt(bidHead) {
			z.Set(bidHead)
		}
	case bidHead.IsZero():
		src := lmp
		if lmp.IsZero() {
			src = askHead
		}
		down, ok := capDown(src, spread)
		if !ok {
			return false
		}
		maxLp(&down)
	default:
		if lmp.IsZero() {
			z.Set(bidHead)
			return true
		}
		down, ok := capDown(lmp, spread)
		if !ok {
			return false
		}
		maxLp(&down)
	}
	return true
}

// bandRate is PoolBands._cappedRate(EngineFee, band.feeMultiplier): the engine's own taker rate
// (the pool's msg.sender on this path) times the band's multiplier, capped at MAX_FEE_RATE.
func (p *PoolSimulator) bandRate(i int) *uint256.Int {
	var rate uint256.Int
	rate.SetUint64(uint64(p.BandPool.EngineFee)*uint64(p.BandPool.Bands[i].FeeMultiplier)).Div(&rate, denom)
	if rate.GtUint64(maxFeeRate) {
		rate.SetUint64(maxFeeRate)
	}
	return &rate
}

type bandWalkResult struct {
	in, out, fee uint256.Int // taken, net paid, fee kept
	matched      uint256.Int // the last reportable band bound (0 if none)
	fills        []bandFill
}

// bandWalk is BandPool._walk + _bandBound + _fillBand for amountIn, tightest band first, each at
// its own bound off the anchor. A Solidity overflow there reverts the swap, which route catches
// and refunds, so an overflow here is an error the caller treats as "no pool leg".
//
// The walk's gas guard (stop before band t > 0 when gasleft() < SWAP_GAS_RESERVE) is not
// modelled: it cannot fire while the executor keeps MATCH_GAS_RESERVE (500,000) above Gas, the
// headroom the book leg already requires.
func (p *PoolSimulator) bandWalk(amountIn *uint256.Int, quoteToBase bool) (*bandWalkResult, error) {
	bp := p.BandPool
	w := &bandWalkResult{}
	limit := bp.LimitSell
	if quoteToBase {
		limit = bp.LimitBuy
	}
	if limit == 0 { // this side is idle for every band
		return w, nil
	}
	var remaining, bound, available, take, out, fee, payout, tmp uint256.Int
	remaining.Set(amountIn)
	for t := range bp.Bands {
		if remaining.IsZero() {
			break
		}
		b := &bp.Bands[t]
		// _tolerance: frac*limit/D, rounded down; zero leaves the band idle.
		tol := uint64(b.SpreadFrac) * uint64(limit) / denom.Uint64()
		if tol == 0 {
			continue
		}
		if quoteToBase {
			// min(ceil(anchor*(D+tol)/D), floor(anchor*(D+limit)/D))
			if _, overflow := bound.MulOverflow(&bp.Anchor, tmp.AddUint64(denom, tol)); overflow {
				return nil, ErrOverflow
			}
			bound.Add(&bound, tmp.SubUint64(denom, 1)).Div(&bound, denom)
			if _, overflow := tmp.MulOverflow(&bp.Anchor, tmp.AddUint64(denom, uint64(limit))); overflow {
				return nil, ErrOverflow
			}
			if tmp.Div(&tmp, denom).Lt(&bound) {
				bound.Set(&tmp)
			}
		} else {
			if tol >= denom.Uint64() {
				continue
			}
			if _, overflow := bound.MulOverflow(&bp.Anchor, tmp.SubUint64(denom, tol)); overflow {
				return nil, ErrOverflow
			}
			bound.Div(&bound, denom)
		}
		if bound.IsZero() {
			continue
		}

		// _fillBand
		payOut := &b.QuoteReserve
		if quoteToBase {
			payOut = &b.BaseReserve
		}
		if !b.HasShares || payOut.IsZero() {
			continue
		}
		if err := convert(&available, &bound, payOut, quoteToBase, p.DecDiff, p.BaseBQuote); err != nil {
			return nil, err
		}
		if available.IsZero() {
			continue
		}
		take.Set(&remaining)
		if available.Lt(&take) {
			take.Set(&available)
		}
		if err := convert(&out, &bound, &take, !quoteToBase, p.DecDiff, p.BaseBQuote); err != nil {
			return nil, err
		}
		if out.IsZero() {
			continue
		}
		if out.Gt(payOut) { // the reserve subtraction would underflow and revert
			return nil, ErrOverflow
		}
		if !out.Lt(payout.Div(payOut, minReportDivisor)) {
			w.matched.Set(&bound)
		}
		if _, overflow := fee.MulOverflow(&out, p.bandRate(t)); overflow {
			return nil, ErrOverflow
		}
		fee.Div(&fee, denom)
		w.in.Add(&w.in, &take)
		w.out.Add(&w.out, tmp.Sub(&out, &fee))
		w.fee.Add(&w.fee, &fee)
		f := bandFill{band: t}
		f.in.Set(&take)
		f.out.Set(&out)
		w.fills = append(w.fills, f)
		remaining.Sub(&remaining, &take)
	}
	return w, nil
}

// report is MatchingLib.reportSwapPrice as PoolFallbackLib._report calls it: the pool's matched
// price, clamped to the market-spread rail around the price the pair opened the block at, becomes
// lmp unless it is zero or unchanged. lmpAtBlockOpen at execution is the snapshot's lmp, which is
// Anchor (the tracker only quotes the pool when nothing has moved lmp since the window).
func (p *PoolSimulator) report(z, lmp, matched *uint256.Int) {
	z.Set(lmp)
	bp := p.BandPool
	if lmp.IsZero() || matched.IsZero() {
		return
	}
	var ceiling, floor, tmp uint256.Int
	if _, overflow := ceiling.MulOverflow(&bp.Anchor, tmp.AddUint64(denom, uint64(bp.RailUp))); overflow {
		return // reportSwapPrice would revert, failing the whole order; never reached by real prices
	}
	ceiling.Div(&ceiling, denom)
	if uint64(bp.RailDown) < denom.Uint64() {
		floor.Mul(&bp.Anchor, tmp.SubUint64(denom, uint64(bp.RailDown))).Div(&floor, denom)
	}
	newLmp := tmp.Set(matched)
	if newLmp.Gt(&ceiling) {
		newLmp.Set(&ceiling)
	}
	if newLmp.Lt(&floor) {
		newLmp.Set(&floor)
	}
	if newLmp.IsZero() {
		return
	}
	z.Set(newLmp)
}
