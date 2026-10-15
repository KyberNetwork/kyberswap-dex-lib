package uscoreprop

import (
	"sort"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// fracBits is the binary fixed-point precision of the dimensionless curve-shape math (2^-64,
// finer than float64's 2^-53). Amounts stay exact integers.
const fracBits = 64

// maxShapeBits bounds shape values so x<<fracBits and products of two stay below 2^255.
const maxShapeBits = 190

var (
	one      = new(uint256.Int).Lsh(big256.U1, fracBits)
	two      = new(uint256.Int).Lsh(big256.U2, fracBits)
	three    = new(uint256.Int).Mul(one, big256.U3)
	maxPoint = big256.UMaxU128
)

// segmentSlopes returns, for the segment ending at each point, the end slopes [left, right] of the
// local cubic through up to 4 neighbouring points (origin included), in normalized segment units
// and clamped to [0, 3] so the Hermite curve stays monotone. Unusable shapes fall back to linear.
func segmentSlopes(points []Point) [][2]uint256.Int {
	slopes := make([][2]uint256.Int, len(points))
	for i := range points {
		if !endSlopes(points, i, &slopes[i][0], &slopes[i][1]) {
			slopes[i][0].Set(one)
			slopes[i][1].Set(one)
		}
		for k := range slopes[i] {
			if s := &slopes[i][k]; s.Sign() < 0 {
				s.Clear()
			} else if s.Gt(three) {
				s.Set(three)
			}
		}
	}
	return slopes
}

// endSlopes differentiates the Newton interpolating polynomial at both ends of segment i.
func endSlopes(points []Point, i int, left, right *uint256.Int) bool {
	at := func(k int) (*uint256.Int, *uint256.Int) {
		if k == 0 {
			return big256.U0, big256.U0
		}
		return &points[k-1][0], &points[k-1][1]
	}
	x0, y0 := at(i)
	x1, y1 := at(i + 1)
	var h, dy, den uint256.Int
	h.Sub(x1, x0)
	if dy.Sub(y1, y0); dy.IsZero() {
		return false
	}
	n := min(4, len(points)+1)
	start := max(0, min(i-1, len(points)+1-n))
	var nodes, coeff [4]uint256.Int
	for k := range n {
		x, y := at(start + k)
		normalize(&nodes[k], x, x0, &h)
		normalize(&coeff[k], y, y0, &dy)
	}
	for order := 1; order < n; order++ {
		for k := n - 1; k >= order; k-- {
			coeff[k].Sub(&coeff[k], &coeff[k-1])
			den.Sub(&nodes[k], &nodes[k-order])
			if !divFix(&coeff[k], &coeff[k], &den) {
				return false
			}
		}
	}
	return derivative(&nodes, &coeff, n, big256.U0, left) && derivative(&nodes, &coeff, n, one, right)
}

// derivative evaluates p'(z) for the Newton form (nodes, coeff) by Horner's rule.
func derivative(nodes, coeff *[4]uint256.Int, n int, z, res *uint256.Int) bool {
	var value, d uint256.Int
	value.Set(&coeff[n-1])
	res.Clear()
	for k := n - 2; k >= 0; k-- {
		d.Sub(z, &nodes[k])
		if !mulFix(res, res, &d) {
			return false
		}
		res.Add(res, &value)
		if !mulFix(&value, &value, &d) {
			return false
		}
		value.Add(&value, &coeff[k])
		if !fits(res) || !fits(&value) {
			return false
		}
	}
	return true
}

// normalize sets z = (v - origin) / scale as signed fixed point; |v - origin| < 2^128.
func normalize(z, v, origin, scale *uint256.Int) {
	if v.Lt(origin) {
		z.Sub(origin, v).Lsh(z, fracBits).Div(z, scale).Neg(z)
		return
	}
	z.Sub(v, origin).Lsh(z, fracBits).Div(z, scale)
}

// mulFix sets z = x*y/2^fracBits (truncated) for two's-complement fixed-point x, y.
func mulFix(z, x, y *uint256.Int) bool {
	var ax, ay uint256.Int
	neg := abs(&ax, x) != abs(&ay, y)
	if ax.BitLen()+ay.BitLen() > fracBits+maxShapeBits {
		return false
	}
	if z.Mul(&ax, &ay).Rsh(z, fracBits); neg {
		z.Neg(z)
	}
	return true
}

// divFix sets z = x*2^fracBits/d (truncated) for two's-complement fixed-point x, d.
func divFix(z, x, d *uint256.Int) bool {
	var ax, ad uint256.Int
	neg := abs(&ax, x) != abs(&ad, d)
	if ad.IsZero() || ax.BitLen() > maxShapeBits {
		return false
	}
	if z.Lsh(&ax, fracBits).Div(z, &ad); z.BitLen() > maxShapeBits {
		return false
	}
	if neg {
		z.Neg(z)
	}
	return true
}

func abs(z, x *uint256.Int) bool {
	if x.Sign() < 0 {
		z.Neg(x)
		return true
	}
	z.Set(x)
	return false
}

func fits(x *uint256.Int) bool {
	var a uint256.Int
	abs(&a, x)
	return a.BitLen() <= maxShapeBits
}

// interpolate sets out to the curve value at amount: exact at sampled points, and between them a
// cubic Hermite segment from y0 to y1 with the precomputed end slopes.
func interpolate(points []Point, slopes [][2]uint256.Int, amount, out *uint256.Int) error {
	if amount.IsZero() {
		return ladder.ErrZeroAmountIn
	}
	if len(points) == 0 {
		return ladder.ErrNoQuote
	}
	i := sort.Search(len(points), func(i int) bool { return !points[i][0].Lt(amount) })
	if i == len(points) {
		return ladder.ErrAmountInTooLarge
	}
	x1, y1 := &points[i][0], &points[i][1]
	if x1.Eq(amount) {
		out.Set(y1)
		return nil
	}
	x0, y0 := big256.U0, big256.U0
	if i > 0 {
		x0, y0 = &points[i-1][0], &points[i-1][1]
	}
	var w, h, u, inner, tmp uint256.Int
	w.Sub(amount, x0)
	h.Sub(x1, x0)
	u.Lsh(&w, fracBits).Div(&u, &h) // in [0, 1)
	left, right := &slopes[i][0], &slopes[i][1]
	// Hermite: out = y0 + dy*u*inner, inner = left + u*(3 - 2*left - right + u*(left + right - 2)).
	// The outer u stays the exact ratio w/h, so a linear segment (inner = 1) is exact to the wei.
	inner.Add(left, right).Sub(&inner, two).Mul(&inner, &u).SRsh(&inner, fracBits)
	inner.Add(&inner, three).Sub(&inner, tmp.Lsh(left, 1)).Sub(&inner, right).Mul(&inner, &u).SRsh(&inner, fracBits)
	if inner.Add(&inner, left).Sign() <= 0 {
		out.Set(y0)
		return nil
	}
	out.Mul(tmp.Sub(y1, y0), &w) // dy*w < 2^256 as both are < 2^128
	out.MulDivOverflow(out, &inner, h.Lsh(&h, fracBits))
	if out.Add(out, y0).Gt(y1) {
		out.Set(y1)
	}
	return nil
}
