package crystalob

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Ported from CrystalMarket._marketOrder / CrystalMath at commit 0c0ecd7 (the source verified on
// monadscan for 0x508254c8...). Only the path the executor uses: exact input, orderType 0
// (partial fill, return the rest), no self-trade prevention, default worst price.

var (
	u1e4 = big256.UBasisPoint
	u1e5 = big256.U100000
)

type orderResult struct {
	amountIn, amountOut, fee uint256.Int // fee is in quote: tokenIn on buys, tokenOut on sells
	si                       SwapInfo
	levels                   int
	usedAMM                  bool
}

// maxFastScaleBits bounds scaleFactor so no solver/AMM product exceeds 256 bits (sizes <= 2^128,
// reserves <= 2^112, fees < 2^17). Below it the solver predicates are monotone; above it the
// literal, wrapping on-chain search is kept.
const maxFastScaleBits = 96

func divUp(z, x, y *uint256.Int) *uint256.Int {
	var rem uint256.Int
	if z.DivMod(x, y, &rem); !rem.IsZero() {
		z.AddUint64(z, 1)
	}
	return z
}

// marketOrder mirrors _marketOrder for an exact-input taker order of origSize.
func (p *PoolSimulator) marketOrder(isBuy bool, origSize *uint256.Int) (r orderResult, err error) {
	if origSize.Gt(big256.UMaxU128) {
		return r, ErrAmountTooLarge
	}
	var size, sizeLeft, tmp uint256.Int
	r.si.IsBuy = isBuy
	amountIn, amountOut := &r.amountIn, &r.amountOut
	worst, levels, endPrice := &p.tickSize, p.bids, big256.U0
	if isBuy {
		divUp(&size, size.Mul(origSize, &p.takerFee), u1e5) // input net of taker fee; < 2^145
		worst, levels, endPrice = &p.buyWorst, p.asks, &p.maxPrice
	} else {
		size.Set(origSize)
	}

	var rq, rb uint256.Int
	ammOn := p.rq != nil && !(p.rq.IsZero() && p.rb.IsZero())
	if ammOn {
		if p.rq.IsZero() || p.rb.IsZero() {
			return r, ErrMarketInactive // on-chain divides by the zero side
		}
		rq.Set(p.rq)
		rb.Set(p.rb)
	}

	li, oi := 0, 0
	for size.Gt(amountIn) {
		sizeLeft.Sub(&size, amountIn)
		price := endPrice
		if li < len(levels) {
			price = levels[li][0]
		}
		beyondWorst := isBuy && price.Gt(worst) || !isBuy && price.Lt(worst)
		if ammOn {
			limit := price
			if beyondWorst {
				limit = worst
			}
			ammIn, ammOut, err := p.ammStep(isBuy, &rq, &rb, limit, &sizeLeft)
			if err != nil {
				return r, err
			} else if !ammOut.IsZero() {
				r.usedAMM = true
				amountIn.Add(amountIn, &ammIn)
				amountOut.Add(amountOut, &ammOut)
				if sizeLeft.Eq(&ammIn) {
					break
				}
				sizeLeft.Sub(&sizeLeft, &ammIn)
			}
		}
		if beyondWorst { // also covers an exhausted book: endPrice is past the default worst price
			break
		}

		r.levels++
		lvl := levels[li]
		for ; 1+oi < len(lvl) && !sizeLeft.IsZero(); oi++ {
			order := lvl[1+oi]
			// Book products fit 256 bits: sizes <= 2^128 (orders 2^112), scale <= 2^112 and
			// price <= 2^80 (CrystalMarket constructor), fees < 2^17.
			tmp.Mul(&sizeLeft, &p.makerRebate).Div(&tmp, u1e5)
			if isBuy {
				tmp.Mul(&tmp, &p.scale).Div(&tmp, price)
			} else {
				tmp.Mul(&tmp, price).Div(&tmp, &p.scale)
			}
			if order.Gt(&tmp) { // _canFillRemaining: partial fill ends the order
				amountOut.Add(amountOut, &tmp)
				r.si.PartialRem = new(uint256.Int).Sub(order, &tmp)
				amountIn.Add(amountIn, &sizeLeft)
				sizeLeft.Clear()
				break
			}
			if isBuy { // maker transfer amount, rounded down
				tmp.Mul(order, price).Div(&tmp, &p.scale)
			} else {
				tmp.Mul(order, &p.scale).Div(&tmp, price)
			}
			tmp.Mul(&tmp, u1e5).Div(&tmp, &p.makerRebate)
			if tmp.Gt(&sizeLeft) {
				return r, ErrOverflow // checked subtraction on-chain
			}
			amountIn.Add(amountIn, &tmp)
			amountOut.Add(amountOut, order)
			sizeLeft.Sub(&sizeLeft, &tmp)
		}
		if r.si.PartialRem == nil && 1+oi >= len(lvl) {
			li, oi = li+1, 0
		}
		if sizeLeft.IsZero() {
			break
		}
	}
	r.si.LevelsDone, r.si.OrdersDone = li, oi

	if amountIn.IsZero() && amountOut.IsZero() {
		return r, ErrInsufficientBook
	}
	if isBuy { // fee in quote: complete fills consume exactly origSize
		r.fee.Set(amountIn)
		if size.Eq(amountIn) {
			amountIn.Set(origSize)
		} else {
			divUp(amountIn, tmp.Mul(amountIn, u1e5), &p.takerFee)
		}
		r.fee.Sub(amountIn, &r.fee)
	} else {
		r.fee.Set(amountOut)
		amountOut.Mul(amountOut, &p.takerFee).Div(amountOut, u1e5) // out < 2^200, fee < 2^17
		r.fee.Sub(&r.fee, amountOut)
	}
	if amountOut.IsZero() {
		return r, ErrInsufficientBook
	}
	if ammOn {
		r.si.ReserveQ, r.si.ReserveB = rq.Clone(), rb.Clone()
	}
	return r, nil
}

// ammStep fills the AMM up to limit (the next book price or the worst price). rq and rb are
// updated in place only when the fill is non-zero.
func (p *PoolSimulator) ammStep(isBuy bool, rq, rb, limit, sizeLeft *uint256.Int) (ammIn, ammOut uint256.Int,
	err error) {
	var num, den uint256.Int
	fast := p.scale.BitLen() <= maxFastScaleBits
	if isBuy {
		if !fast && mulOverflows(rq, &p.scale, u1e4, &p.makerRebate) {
			return ammIn, ammOut, ErrOverflow // checked math on-chain
		}
		num.Mul(rq, &p.scale).Mul(&num, u1e4).Mul(&num, &p.makerRebate)
		den.Mul(rb, &p.ammFee).Mul(&den, u1e5)
		if !limit.Gt(divUp(&num, &num, &den)) {
			return
		}
		ammIn = exactInputBuySolve(rq, rb, limit, &p.makerRebate, sizeLeft, &p.scale, &p.ammFee, fast)
		num.Mul(&ammIn, &p.ammFee)
		den.Mul(rq, u1e4).Add(&den, &num)
		ammOut.Mul(&num, rb).Div(&ammOut, &den) // ammIn*fee < 2^142, reserve <= 2^112
		if ammOut.IsZero() {
			return
		}
		rq.Add(rq, &ammIn)
		rb.Sub(rb, &ammOut)
	} else {
		if !fast && mulOverflows(rq, &p.scale, &p.ammFee, u1e5) {
			return ammIn, ammOut, ErrOverflow
		}
		num.Mul(rq, &p.scale).Mul(&num, &p.ammFee).Mul(&num, u1e5)
		den.Mul(rb, u1e4).Mul(&den, &p.makerRebate)
		if !limit.Lt(num.Div(&num, &den)) {
			return
		}
		ammIn = exactInputSellSolve(rq, rb, limit, &p.makerRebate, sizeLeft, &p.scale, &p.ammFee, fast)
		num.Mul(&ammIn, &p.ammFee)
		den.Mul(rb, u1e4).Add(&den, &num)
		ammOut.Mul(&num, rq).Div(&ammOut, &den)
		if ammOut.IsZero() {
			return
		}
		rb.Add(rb, &ammIn)
		rq.Sub(rq, &ammOut)
	}
	if rq.Gt(maskU112) || rb.Gt(maskU112) {
		err = ErrOverflow
	}
	return
}

func mulOverflows(a, b, c, d *uint256.Int) bool {
	var z uint256.Int
	_, o1 := z.MulOverflow(a, b)
	_, o2 := z.MulOverflow(&z, c)
	_, o3 := z.MulOverflow(&z, d)
	return o1 || o2 || o3
}

// exactInputBuySolve is CrystalMath._exactInputBuySolve: the largest input in [0, high] whose
// post-swap price stays <= target. With fast set the price is monotone in the input, so a passing
// high skips the search with the same result.
func exactInputBuySolve(rq, rb, target, mr, high, s, fee *uint256.Int, fast bool) (low uint256.Int) {
	var hi, p uint256.Int
	if high.IsZero() {
		return
	} else if fast && !buyPriceAfter(&p, high, rq, rb, mr, s, fee).Gt(target) {
		return *high
	}
	hi.Set(high)
	var mid uint256.Int
	for low.Lt(&hi) {
		mid.Sub(&hi, &low).AddUint64(&mid, 1).Rsh(&mid, 1).Add(&mid, &low)
		if buyPriceAfter(&p, &mid, rq, rb, mr, s, fee).Gt(target) {
			hi.SubUint64(&mid, 1)
		} else {
			low.Set(&mid)
		}
	}
	return
}

// buyPriceAfter is the solver's pMid for input mid (unchecked, wraps like the contract).
func buyPriceAfter(z, mid, rq, rb, mr, s, fee *uint256.Int) *uint256.Int {
	var t, den uint256.Int
	t.Mul(mid, fee)
	den.Mul(rq, u1e4).Add(&den, &t)
	t.Mul(&t, rb).Div(&t, &den)
	den.Sub(rb, &t).Mul(&den, fee).Mul(&den, u1e5)
	z.Add(rq, mid).Mul(z, u1e4).Mul(z, s).Mul(z, mr)
	return divUp(z, z, &den)
}

// exactInputSellSolve is CrystalMath._exactInputSellSolve: the largest input in [0, high] whose
// post-swap price stays >= target.
func exactInputSellSolve(rq, rb, target, mr, high, s, fee *uint256.Int, fast bool) (low uint256.Int) {
	var hi, p uint256.Int
	if high.IsZero() {
		return
	} else if fast && !sellPriceAfter(&p, high, rq, rb, mr, s, fee).Lt(target) {
		return *high
	}
	hi.Set(high)
	var mid uint256.Int
	for low.Lt(&hi) {
		mid.Sub(&hi, &low).AddUint64(&mid, 1).Rsh(&mid, 1).Add(&mid, &low)
		if sellPriceAfter(&p, &mid, rq, rb, mr, s, fee).Lt(target) {
			hi.SubUint64(&mid, 1)
		} else {
			low.Set(&mid)
		}
	}
	return
}

// sellPriceAfter is the sell solver's pMid for input mid.
func sellPriceAfter(z, mid, rq, rb, mr, s, fee *uint256.Int) *uint256.Int {
	var t, den uint256.Int
	t.Mul(mid, fee)
	den.Mul(rb, u1e4).Add(&den, &t)
	t.Mul(&t, rq).Div(&t, &den)
	z.Sub(rq, &t).Mul(z, fee).Mul(z, s).Mul(z, u1e5)
	den.Add(rb, mid).Mul(&den, u1e4).Mul(&den, mr)
	return z.Div(z, &den)
}

// defaultBuyWorstPrice is one tick below maxPrice, as _marketOrder defaults it for taker buys.
func defaultBuyWorstPrice(marketType uint64, tickSize, maxPrice *uint256.Int) (*uint256.Int, bool) {
	if marketType == 0 {
		return new(uint256.Int).Sub(maxPrice, tickSize), true
	}
	var p uint256.Int
	if !p.Div(maxPrice, tickSize).IsUint64() {
		return nil, false
	}
	tick, ok := priceToTick(p.Uint64())
	if !ok || tick == 0 {
		return nil, false
	}
	return tickToPrice(tick-1, tickSize), true
}

// priceToTick is CrystalMath._priceToTick on p = price / tickSize.
func priceToTick(p uint64) (uint64, bool) {
	if p <= 100_000 {
		return p, true
	}
	for lim, d, base := uint64(1_000_000), uint64(10), uint64(90_000); lim <= 1e15; lim, d, base = lim*10, d*10,
		base+90_000 {
		if p < lim || lim == 1e15 && p == lim {
			return base + p/d, p%d == 0
		}
	}
	return 0, false
}

// tickToPrice is CrystalMath._tickToPrice.
func tickToPrice(t uint64, tickSize *uint256.Int) *uint256.Int {
	if t <= 100_000 {
		return new(uint256.Int).Mul(uint256.NewInt(t), tickSize)
	}
	x := t - 10_000
	res := new(uint256.Int).Set(big256.TenPow(x / 90_000)) // TenPow returns a shared value
	return res.Mul(res, uint256.NewInt(10_000+x%90_000)).Mul(res, tickSize)
}
