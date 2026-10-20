package pancakeprop

import (
	"github.com/holiman/uint256"
)

// The pricing implementation behind PropAMMEntry.pricing() is unverified on both explorers and
// Sourcify. This file ports its quoteExactInput from the deployed bytecode (traced opcode by
// opcode on a fork), and was differentially fuzzed against eth_call.

var (
	u2e17  = uint256.NewInt(200_000_000_000_000_000)              // spread denominator: spread / 2e17 is the half-spread
	uScale = uint256.MustFromDecimal("1000000000000000000000000") // mid price scale 1e24
	uWad   = uint256.NewInt(1_000_000_000_000_000_000)
	u1e17  = uint256.NewInt(100_000_000_000_000_000)
	uBps   = uint256.NewInt(10_000)
	u1e10  = uint256.NewInt(10_000_000_000)
)

// price is the unpacked per-pair price word (bit layout recovered from the bytecode):
// [10,50) expiry block, [50,106) mid mantissa, [106,114) mid exponent, [145,170) base spread,
// [170,195) max spread (both x1e10), 195 concave, [196,200) growth, [200,214) and [214,228)
// max capacity ratio (token0 / token1 input), [228,242) and [242,256) flat ratio.
type price struct {
	expiry               uint64
	mid, base, maxSpread uint256.Int
	concave              bool
	growth               uint64
	ratioMax, ratioFlat  [2]uint64
	midErr               bool // exponent > 200, or exponent != 0 with mantissa < 2^55
}

func field(w *uint256.Int, lo uint, bits uint) uint64 {
	var t uint256.Int
	t.Rsh(w, lo)
	return t.Uint64() & (1<<bits - 1)
}

func unpackPrice(w *uint256.Int) *price {
	p := &price{
		expiry:  field(w, 10, 40),
		concave: field(w, 195, 1) == 1,
		growth:  field(w, 196, 4),
	}
	mant, exp := field(w, 50, 56), field(w, 106, 8)
	p.midErr = exp > 200 || (exp != 0 && mant < 1<<55)
	p.mid.SetUint64(mant)
	p.mid.Lsh(&p.mid, uint(exp))
	p.base.SetUint64(field(w, 145, 25))
	p.base.Mul(&p.base, u1e10)
	p.maxSpread.SetUint64(field(w, 170, 25))
	p.maxSpread.Mul(&p.maxSpread, u1e10)
	for d := range 2 {
		p.ratioMax[d] = field(w, uint(200+14*d), 14)
		p.ratioFlat[d] = field(w, uint(228+14*d), 14)
	}
	return p
}

// mulDiv is OpenZeppelin Math.mulDiv: floor(x*y/d), reverting on d == 0 or a result above 2^256.
func mulDiv(z, x, y, d *uint256.Int) error {
	if d.IsZero() {
		return ErrOverflow
	}
	if _, overflow := z.MulDivOverflow(x, y, d); overflow {
		return ErrOverflow
	}
	return nil
}

// mulDivUp is Math.mulDiv with Rounding.Ceil.
func mulDivUp(z, x, y, d *uint256.Int) error {
	var rem uint256.Int
	if d.IsZero() {
		return ErrOverflow
	}
	rem.MulMod(x, y, d)
	if err := mulDiv(z, x, y, d); err != nil {
		return err
	}
	if !rem.IsZero() {
		if _, overflow := z.AddOverflow(z, uint256.NewInt(1)); overflow {
			return ErrOverflow
		}
	}
	return nil
}

// ceilMulDiv returns ceil(x*y/d) for x*y < 2^256 (the caller states the bound).
func ceilMulDiv(z, x, y, d *uint256.Int) {
	var t, rem uint256.Int
	t.Mul(x, y)
	z.DivMod(&t, d, &rem)
	if !rem.IsZero() {
		z.AddUint64(z, 1)
	}
}

// out prices x of input at the given spread: bid side divides, ask side multiplies by mid.
func (p *price) out(z *uint256.Int, fwd bool, x, spread *uint256.Int) error {
	var pr, t uint256.Int
	if fwd {
		t.Sub(u2e17, spread)
		if err := mulDiv(&pr, &p.mid, &t, u2e17); err != nil {
			return err
		}
		return mulDiv(z, x, &pr, uScale)
	}
	t.Add(u2e17, spread)
	if err := mulDivUp(&pr, &p.mid, &t, u2e17); err != nil {
		return err
	}
	return mulDiv(z, x, uScale, &pr)
}

// quote ports quoteExactInput for one direction. fwd is true when the input is token0, maxInput
// is that direction's pair-level capacity, cum its epoch cumulative input (both < 2^128). The
// expiry, vault balance and pause checks belong to the caller.
func (p *price) quote(fwd bool, maxInput, cum, amountIn *uint256.Int) (res uint256.Int, err error) {
	dir := 0
	if !fwd {
		dir = 1
	}
	rMax, rFlat := p.ratioMax[dir], p.ratioFlat[dir]

	if p.midErr {
		return res, ErrInvalidMidPrice
	}
	if !p.base.Lt(u2e17) || !p.maxSpread.Lt(u2e17) {
		return res, ErrInvalidSpread
	}
	var t uint256.Int
	t.Sub(u2e17, &p.maxSpread)
	if _, overflow := t.MulOverflow(&p.mid, &t); overflow {
		return res, ErrOverflow
	}
	if t.Lt(u2e17) {
		return res, ErrInvalidMidPrice
	}
	if p.maxSpread.Lt(&p.base) {
		return res, ErrInvalidDepth
	}
	if p.growth == 0 || (p.concave && p.growth > 9) {
		return res, ErrInvalidGrowth
	}
	if rMax > 10_000 || rFlat > rMax {
		return res, ErrInvalidDepth
	}

	var maxCum, flatCum, newCum, ratio uint256.Int
	if err = mulDiv(&maxCum, maxInput, ratio.SetUint64(rMax), uBps); err != nil {
		return
	}
	if err = mulDiv(&flatCum, maxInput, ratio.SetUint64(rFlat), uBps); err != nil {
		return
	}
	if _, overflow := newCum.AddOverflow(cum, amountIn); overflow {
		return res, ErrOverflow
	}
	if newCum.Gt(&maxCum) {
		return res, ErrExceedsCapacity
	}

	// Split the input into the flat part (priced at the base spread) and the curve part.
	var x, a uint256.Int
	if cum.Lt(&flatCum) {
		var fi uint256.Int
		fi.Sub(&flatCum, cum)
		if fi.Gt(amountIn) {
			fi.Set(amountIn)
		}
		if err = p.out(&res, fwd, &fi, &p.base); err != nil {
			return
		}
		x.Sub(amountIn, &fi)
	} else {
		x.Set(amountIn)
		a.Sub(cum, &flatCum)
	}
	if x.IsZero() {
		return
	}

	var l, ds, avg, spread, curveOut uint256.Int
	l.Sub(&maxCum, &flatCum) // > 0: x > 0 and newCum <= maxCum
	ds.Sub(&p.maxSpread, &p.base)
	if p.concave {
		err = p.concaveAvg(&avg, &ds, &l, &a, &x)
	} else {
		p.convexAvg(&avg, &ds, &l, &a, &x)
	}
	if err != nil {
		return
	}
	if avg.Gt(&ds) {
		avg.Set(&ds)
	}
	spread.Add(&p.base, &avg)
	if err = p.out(&curveOut, fwd, &x, &spread); err != nil {
		return
	}
	if _, overflow := res.AddOverflow(&res, &curveOut); overflow {
		return res, ErrOverflow
	}
	return
}

// convexAvg is the average extra spread over [a, a+x] of the curve ds*(u/l)^growth: each of the
// growth+1 power steps rounds up for the upper bound b and down for the lower bound a.
// Bounds: ds < 2^58 and a, b, l < 2^129, so every product below fits in 256 bits.
func (p *price) convexAvg(avg, ds, l, a, x *uint256.Int) {
	var b, tb, ta, t, n uint256.Int
	b.Add(a, x)
	tb.Set(ds)
	ta.Set(ds)
	for range p.growth + 1 {
		ceilMulDiv(&tb, &tb, &b, l)
		t.Mul(&ta, a)
		ta.Div(&t, l)
	}
	t.Sub(&tb, &ta) // tb >= ta since b >= a
	n.SetUint64(p.growth + 1)
	n.Mul(&n, x)
	ceilMulDiv(avg, &t, l, &n)
}

// concaveAvg is the same average for the curve ds*(u/l)^(growth/10), integrated through
// powWad(u/l, 1+growth/10) / (1+growth/10).
func (p *price) concaveAvg(avg, ds, l, a, x *uint256.Int) error {
	var b, wb, wa, pb, pa, y, t, d uint256.Int
	b.Add(a, x)
	y.SetUint64(p.growth + 10)
	// wad ratios: b*1e18 < 2^129 * 2^60
	ceilMulDiv(&wb, &b, uWad, l)
	t.Mul(a, uWad)
	wa.Div(&t, l)

	y.Mul(&y, u1e17)
	if err := powWad(&pb, &wb, &y); err != nil {
		return err
	}
	if !wa.IsZero() {
		if err := powWad(&pa, &wa, &y); err != nil {
			return err
		}
	}
	if pb.Lt(&pa) {
		return ErrOverflow
	}
	d.Sub(&pb, &pa)
	d.Mul(&d, uint256.NewInt(10)) // d <= ~1e18
	y.SetUint64(p.growth + 10)
	y.Mul(&y, uWad)
	ceilMulDiv(&t, ds, &d, &y) // ds < 2^58, d < 2^64
	ceilMulDiv(avg, &t, l, x)
	return nil
}
