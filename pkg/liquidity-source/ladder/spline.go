package ladder

import (
	"math"
	"sort"
)

// Spline estimates amountOut for an arbitrary amountIn from a small set of
// on-chain-probed (amountIn, amountOut) points, blending two monotone cubic
// Hermite fits:
//   - a primary fit directly over (amountIn, amountOut), plus an implicit
//     (0,0) anchor so amounts below the first real sample interpolate
//     smoothly toward the origin instead of needing a separate prorated
//     special case;
//   - a capacity-space fit over (ln(amountIn), ln(capacityCeiling-amountOut))
//     (see capacity.go) that sharpens quotes right before a reserve -
//     depletion cliff a sparse geometric sample grid can miss entirely in
//     the primary fit.
//
// Both fits use monotone Hermite instead of a plain/natural cubic spline
// because it cannot overshoot between points -- important here since a swap
// curve must never imply a negative or decreasing amountOut for an
// increasing amountIn, which an unconstrained spline can produce when points
// are unevenly spaced.
type Spline struct {
	x, y, tangent []float64

	// Capacity-space fit fields; see capacity.go. capacityCeiling stays 0
	// (its zero value) when there are too few points to fit a curve, which
	// QuoteAmountOut treats as "no capacity fit available".
	capacityCeiling      float64
	capacityLogX         []float64
	capacityLogRemaining []float64
	capacityTangent      []float64
}

// NewSpline builds the spline from points sorted ascending by AmountIn.
func NewSpline(points []Point) *Spline {
	n := len(points)
	x := make([]float64, n+1)
	y := make([]float64, n+1)
	for i, p := range points {
		x[i+1], y[i+1] = p[0], p[1]
	}
	s := &Spline{x: x, y: y, tangent: amountOutTangents(x, y)}
	s.buildCapacityFit(points)
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
	h := s.x[hi] - s.x[lo]
	t := (amountIn - s.x[lo]) / h
	t2, t3 := t*t, t*t*t

	h00 := 2*t3 - 3*t2 + 1
	h10 := t3 - 2*t2 + t
	h01 := -2*t3 + 3*t2
	h11 := t3 - t2

	out := h00*s.y[lo] + h10*h*s.tangent[lo] + h01*s.y[hi] + h11*h*s.tangent[hi]

	if s.capacityCeiling == 0 {
		return out, nil
	}
	remaining := (s.capacityCeiling - out) / s.capacityCeiling
	if remaining >= capacityBlendHi {
		return out, nil
	}
	capacityOut, ok := s.quoteCapacitySpace(amountIn)
	if !ok {
		return out, nil
	}
	w := 1.0
	if remaining > capacityBlendLo {
		w = (capacityBlendHi - remaining) / (capacityBlendHi - capacityBlendLo)
	}
	return (1-w)*out + w*capacityOut, nil
}

// amountOutTangents is pchipTangents plus a never-overquote floor.
//
// For a Hermite segment [lo,hi] with secant S, p(t) <= chord(t) for every t
// in the segment iff m_lo <= S and m_hi >= S (p(t)-chord(t) factors as
// h*t*(t-1)*[S(1-2t)+m_lo(t-1)+m_hi*t], linear in t, so the sign only needs
// checking at t=0,1: S-m_lo and m_hi-S must both be >= 0). At a decelerating
// node (secant[i-1] > secant[i]) no single tangent can satisfy
// "m_hi >= secant[i-1]" for the segment before it AND "m_lo <= secant[i]"
// for the segment after -- they're mutually exclusive. We pick protecting
// the segment before the knee (flooring to the steeper neighbor), since
// it's the one that showed real-world overquote risk; the segment after
// tends to already undershoot the real curve (a reserve-cap cliff, further
// sharpened by capacityTangents in capacity.go), so losing its guarantee
// here costs little. Applied before fcMonotonicitySafety, unlike
// capacityTangents' sandwich clamp, so that safety pass can still rein in
// any resulting non-monotonicity from the floor.
func amountOutTangents(x, y []float64) []float64 {
	tangent, secant := pchipTangents(x, y)
	if len(secant) == 0 {
		return tangent
	}

	for i := 1; i < len(tangent)-1; i++ {
		if secant[i-1] > secant[i] {
			tangent[i] = math.Max(tangent[i], secant[i-1])
		}
	}

	fcMonotonicitySafety(tangent, secant)

	return tangent
}

// QuoteAmountOut is a convenience one-shot wrapper around NewSpline, for
// callers that don't need to query the same curve repeatedly. Repeated
// callers (e.g. PoolSimulator, quoting many times per route) should build a
// *Spline once instead.
func QuoteAmountOut(points []Point, amountIn float64) (float64, error) {
	return NewSpline(points).QuoteAmountOut(amountIn)
}
