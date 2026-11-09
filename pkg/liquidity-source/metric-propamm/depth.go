package metricpropamm

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// walkDepth is Metric's exact-input walk (docs: "Estimate a fill from depth"): cross bins whole,
// then price the bin the order ends in with fillInside. big.Int because the in-bin products exceed 512 bits.
func (s *PoolSimulator) walkDepth(asks bool, bins []Bin, amountIn *big.Int) (*big.Int, error) {
	var remaining, received, prevIn, prevOut, binIn, binOut big.Int
	remaining.Set(amountIn)
	for _, bin := range bins {
		if bin.CumulativeInput.Cmp(&prevIn) <= 0 {
			continue
		}
		binIn.Sub(bin.CumulativeInput, &prevIn)
		binOut.Sub(bin.CumulativeVolume, &prevOut)
		prevIn.Set(bin.CumulativeInput)
		prevOut.Set(bin.CumulativeVolume)
		if binOut.Sign() == 0 {
			continue
		}
		if remaining.Cmp(&binIn) < 0 {
			return received.Add(&received, s.fillInside(asks, bin.Price, &binIn, &binOut, &remaining)), nil
		}
		received.Add(&received, &binOut)
		if remaining.Sub(&remaining, &binIn).Sign() == 0 {
			return &received, nil
		}
	}
	return nil, ErrInsufficientLiquidity
}

// fillInside returns the output for input r (< I) inside a bin of input I, output O and closing price p.
// It never exceeds what the pool pays: the doc's safety margin is subtracted.
func (s *PoolSimulator) fillInside(asks bool, p, I, O, r *big.Int) *big.Int {
	var prop, closeIn, E, t, u, out big.Int
	prop.Mul(O, r).Div(&prop, I)

	// closeIn: whole bin priced at the closing price. 10^d1/10^d0 = 10^-decimalsDiff, one floor.
	if asks {
		t.Mul(O, p)
		u.Lsh(bignumber.One, 64)
		s.scaleDiv(&t, &u, -s.decimalsDiff)
	} else {
		t.Lsh(O, 64)
		u.Set(p)
		s.scaleDiv(&t, &u, s.decimalsDiff)
	}
	closeIn.Div(&t, &u)
	E.Sub(&closeIn, I)
	if E.Sign() <= 0 || (asks && E.Cmp(I) > 0) || (!asks && E.Cmp(I) >= 0) {
		return &prop
	}

	var iMinusE, iPlusE, low big.Int
	iMinusE.Sub(I, &E)
	iPlusE.Add(I, &E)
	low.Set(p)
	if asks {
		low.Mul(p, &iMinusE).Div(&low, &iPlusE)
		if low.Sign() == 0 {
			return &prop
		}
	}

	if asks {
		// solve (I-E)x + E x^2 = r with 64 fractional bits, root rounded up
		var sq big.Int
		sq.Lsh(&iMinusE, 64) // S
		t.Mul(&sq, &sq)
		u.Mul(&E, r).Lsh(&u, 130)
		t.Add(&t, &u)
		ceilSqrt(&t, &t)
		t.Add(&t, &sq)
		out.Mul(O, r).Lsh(&out, 65).Div(&out, &t)
	} else {
		var g, bi big.Int
		g.Mul(O, r).Div(&g, &iMinusE)
		if g.Cmp(O) > 0 {
			g.Set(O)
		}
		bidIn(&bi, &g, I, O, &E)
		if bi.Cmp(r) <= 0 {
			out.Set(&g)
		} else {
			out.Mul(&g, r).Div(&out, &bi)
		}
	}

	out.Sub(&out, margin(&out, &low))
	if out.Cmp(&prop) < 0 {
		out.Set(&prop)
	}
	if out.Cmp(O) > 0 {
		out.Set(O)
	}
	return &out
}

// scaleDiv rewrites t/u as t*10^exp/u: multiplies t for exp>0, u for exp<0.
func (s *PoolSimulator) scaleDiv(t, u *big.Int, exp int) {
	if exp > 0 {
		t.Mul(t, bignumber.TenPowInt(exp))
	} else if exp < 0 {
		u.Mul(u, bignumber.TenPowInt(-exp))
	}
}

// bidIn is the bids input charged for output o: ceil(o(I-E)((I+E)O-Eo) / (O((I+E)O-2Eo))).
func bidIn(res, o, I, O, E *big.Int) {
	var iMinusE, iPlusEO, num, den, t big.Int
	iMinusE.Sub(I, E)
	iPlusEO.Add(I, E).Mul(&iPlusEO, O)
	t.Mul(E, o)
	num.Sub(&iPlusEO, &t)
	den.Sub(&num, &t).Mul(&den, O)
	num.Mul(&num, &iMinusE).Mul(&num, o)
	bignumber.MulDivUp(res, &num, bignumber.One, &den)
}

// margin(a) = ceil(2a/low) + 1.
func margin(a, low *big.Int) *big.Int {
	var m big.Int
	m.Lsh(a, 1)
	bignumber.MulDivUp(&m, &m, bignumber.One, low)
	return m.Add(&m, bignumber.One)
}

// ceilSqrt sets z = ceil(sqrt(x)).
func ceilSqrt(z, x *big.Int) {
	var r, sq big.Int
	r.Sqrt(x)
	if sq.Mul(&r, &r).Cmp(x) < 0 {
		r.Add(&r, bignumber.One)
	}
	z.Set(&r)
}
