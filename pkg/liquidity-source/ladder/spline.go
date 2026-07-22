package ladder

import (
	"math"
	"sort"
)

// bezierWeight is the rational quadratic Bezier's control-point weight (see
// NewSpline). 1.0 would make it a plain (non-rational) quadratic Bezier;
// pulling it above 1 pulls the curve closer to the control point (the
// corner formed by the two neighboring segments' secants), tightening the
// fit right at a knee without risking overquote -- verified against real
// on-chain ladders up to weight 7 before crossing into overquote territory,
// so 1.6 leaves comfortable margin.
const bezierWeight = 1.6

// Spline estimates amountOut for an arbitrary amountIn from a small set of
// on-chain-probed (amountIn, amountOut) points, using a rational quadratic
// Bezier curve per segment, plus an implicit (0,0) anchor so amounts below
// the first real sample interpolate smoothly toward the origin instead of
// needing a separate pro-rated special case.
//
// For the segment [x[i], x[i+1]], the curve's control point is the
// intersection of two lines: one through (x[i], y[i]) with the slope of the
// PRECEDING segment's own secant, and one through (x[i+1], y[i+1]) with the
// slope of the FOLLOWING segment's own secant -- i.e. the curve is tangent
// to each neighbor's trend, rather than to some blend/average involving
// this segment's own secant the way a monotone cubic Hermite spline (e.g.
// Fritsch-Carlson) does. A Bezier curve is provably bounded within the
// convex hull of its control points, so it cannot overshoot past where
// those two extrapolated lines cross, the way an unconstrained cubic
// Hermite can when the neighboring secants are very different from this
// segment's own.
//
// That boundedness is also what lets this handle a reserve-depletion cliff
// correctly on its own, with no separate capacity-space fit blended in near
// the cap: the following segment's secant collapsing toward zero as the
// ladder approaches its sampled plateau is exactly the signal a knee is
// near, and the intersection construction bends the curve toward the
// plateau accordingly.
//
// One safety clamp is required: decelerating segments (secant dropping
// going into the next one -- the common "approaching a knee" shape)
// naturally keep the intersection point inside this segment's own
// [x_lo,x_hi] x [y_lo,y_hi] box, but an accelerating segment can push the
// intersection outside that box, which breaks monotonicity. Clamping the
// control point back into the box (see NewSpline) fixes that.
type Spline struct {
	x, y               []float64
	controlX, controlY []float64 // one Bezier control point per segment
}

// NewSpline builds the spline from points sorted ascending by AmountIn.
func NewSpline(points []Point) *Spline {
	n := len(points)
	x := make([]float64, n+1)
	y := make([]float64, n+1)
	for i, p := range points {
		x[i+1], y[i+1] = p[0], p[1]
	}

	s := &Spline{x: x, y: y}
	m := len(x) - 1
	if m < 1 {
		return s
	}

	secant := make([]float64, m)
	for i := range m {
		secant[i] = (y[i+1] - y[i]) / (x[i+1] - x[i])
	}

	s.controlX = make([]float64, m)
	s.controlY = make([]float64, m)
	for i := range m {
		xLo, yLo, xHi, yHi := x[i], y[i], x[i+1], y[i+1]
		secantLo, secantHi := secant[i], secant[i]
		if i > 0 {
			secantLo = secant[i-1]
		}
		if i+1 < m {
			secantHi = secant[i+1]
		}

		if secantLo == secantHi {
			// Degenerate (parallel outer lines, or no neighbors on either
			// side): the chord's own midpoint is colinear with its
			// endpoints, which makes the curve exactly the chord.
			s.controlX[i] = (xLo + xHi) / 2
			s.controlY[i] = (yLo + yHi) / 2
			continue
		}

		xInt := (yHi - secantHi*xHi - yLo + secantLo*xLo) / (secantLo - secantHi)
		yInt := yLo + secantLo*(xInt-xLo)

		s.controlX[i] = math.Max(xLo, math.Min(xHi, xInt))
		s.controlY[i] = math.Max(yLo, math.Min(yHi, yInt))
	}

	return s
}

// QuoteAmountOut evaluates the spline at amountIn. Beyond the last probed
// point the curve is unknown, so ErrAmountInTooLarge is returned rather than
// extrapolating.
func (s *Spline) QuoteAmountOut(amountIn float64) (float64, error) {
	if amountIn <= 0 {
		return 0, ErrZeroAmountIn
	} else if len(s.x) <= 1 {
		return 0, ErrNoQuote
	}

	i := sort.Search(len(s.x), func(j int) bool { return s.x[j] >= amountIn })
	if i == len(s.x) {
		return 0, ErrAmountInTooLarge
	} else if s.x[i] == amountIn {
		return s.y[i], nil
	}

	lo, hi := i-1, i
	if s.y[lo] == s.y[hi] {
		// Flat segment (e.g. past the reserve cap): return the constant
		// directly rather than let the Bezier math round-trip it, which can
		// introduce last-bit floating-point jitter that trips strict
		// monotonicity checks despite being mathematically exact.
		return s.y[lo], nil
	}

	t := solveBezierT(s.x[lo], s.controlX[lo], s.x[hi], amountIn)
	return rationalBezier(s.y[lo], s.controlY[lo], s.y[hi], t), nil
}

// solveBezierT finds t in [0,1] such that the rational quadratic Bezier
// x(t) (with control point xInt, weight bezierWeight) equals xj. Clearing
// the rational form's denominator leaves a plain quadratic in t.
func solveBezierT(xLo, xInt, xHi, xj float64) float64 {
	w := bezierWeight
	a := (xLo - xj) - 2*w*(xInt-xj) + (xHi - xj)
	b := -2*(xLo-xj) + 2*w*(xInt-xj)
	c := xLo - xj

	var t float64
	if math.Abs(a) < 1e-12 {
		if b != 0 {
			t = -c / b
		}
	} else {
		disc := math.Max(b*b-4*a*c, 0)
		sqrtDisc := math.Sqrt(disc)
		t1 := (-b + sqrtDisc) / (2 * a)
		if t1 >= 0 && t1 <= 1 {
			t = t1
		} else {
			t = (-b - sqrtDisc) / (2 * a)
		}
	}
	return math.Max(0, math.Min(1, t))
}

// rationalBezier evaluates the rational quadratic Bezier curve (with weight
// bezierWeight) through (t=0 -> yLo, t=1 -> yHi, control value yInt) at t.
func rationalBezier(yLo, yInt, yHi, t float64) float64 {
	w := bezierWeight
	u := 1 - t
	num := u*u*yLo + 2*w*u*t*yInt + t*t*yHi
	den := u*u + 2*w*u*t + t*t
	return num / den
}

// QuoteAmountOut is a convenience one-shot wrapper around NewSpline, for
// callers that don't need to query the same curve repeatedly. Repeated
// callers (e.g. PoolSimulator, quoting many times per route) should build a
// *Spline once instead.
func QuoteAmountOut(points []Point, amountIn float64) (float64, error) {
	return NewSpline(points).QuoteAmountOut(amountIn)
}
