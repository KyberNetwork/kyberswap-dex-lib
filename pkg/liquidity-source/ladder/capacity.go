package ladder

import (
	"math"
	"sort"
)

// capacityCeilingSlack pads capacityCeiling (the highest sampled amountOut)
// by a small margin so ln(capacityCeiling-amountOut) stays finite at that
// top sample, where amountOut equals the raw max exactly.
const capacityCeilingSlack = 1.0001

// capacityBlendHi/Lo bound how much of the capacity-space estimate gets
// blended into the primary one, based on the fraction of capacityCeiling the
// primary estimate implies is still unused: at or above capacityBlendHi
// remaining, the primary fit is trusted alone (it's accurate away from a
// cliff); at or below capacityBlendLo, the capacity-space fit is trusted
// alone; linear blend in between.
const (
	capacityBlendHi = 0.10
	capacityBlendLo = 0.03
)

// buildCapacityFit fits a second monotone Hermite curve over
// (ln(amountIn), ln(capacityCeiling-amountOut)) -- "remaining capacity" --
// instead of (amountIn, amountOut) directly. Right before a reserve-
// depletion cliff, amountOut approaches its ceiling asymptotically; fitting
// the remaining capacity in log-log space captures that shape far better
// than fitting amountOut itself, where the same cliff looks like a
// near-discontinuity a sparse geometric sample grid can miss entirely.
// QuoteAmountOut only consults this once the primary fit's own estimate
// says little capacity is left (see quoteCapacitySpace), so it costs
// nothing in the common case.
//
// It's a no-op (capacityCeiling stays 0, so QuoteAmountOut skips straight to
// the primary fit) when there are too few points to define a ceiling or a
// curve.
func (s *Spline) buildCapacityFit(points []Point) {
	n := len(points)
	if n < 2 {
		return
	}

	capacityCeiling := 0.0
	for _, p := range points {
		capacityCeiling = math.Max(capacityCeiling, p.AmountOut())
	}
	capacityCeiling *= capacityCeilingSlack

	logX := make([]float64, n)
	logRemaining := make([]float64, n)
	for i, p := range points {
		logX[i] = math.Log(p.AmountIn())
		logRemaining[i] = math.Log(capacityCeiling - p.AmountOut())
	}

	s.capacityCeiling = capacityCeiling
	s.capacityLogX = logX
	s.capacityLogRemaining = logRemaining
	s.capacityTangent = capacityTangents(logX, logRemaining)
}

// quoteCapacitySpace evaluates the capacity-space fit at amountIn and maps
// it back to amountOut. ok is false outside the fitted range (no ceiling
// built, or amountIn below the first / at-or-beyond the last sample), in
// which case the caller should fall back to the primary fit alone.
func (s *Spline) quoteCapacitySpace(amountIn float64) (out float64, ok bool) {
	if s.capacityCeiling == 0 {
		return 0, false
	}
	logIn := math.Log(amountIn)
	i := sort.Search(len(s.capacityLogX), func(j int) bool { return s.capacityLogX[j] >= logIn })
	if i == 0 || i == len(s.capacityLogX) {
		return 0, false
	}

	lo, hi := i-1, i
	h := s.capacityLogX[hi] - s.capacityLogX[lo]
	t := (logIn - s.capacityLogX[lo]) / h
	t2, t3 := t*t, t*t*t

	h00 := 2*t3 - 3*t2 + 1
	h10 := t3 - 2*t2 + t
	h01 := -2*t3 + 3*t2
	h11 := t3 - t2

	logRemaining := h00*s.capacityLogRemaining[lo] + h10*h*s.capacityTangent[lo] +
		h01*s.capacityLogRemaining[hi] + h11*h*s.capacityTangent[hi]
	return s.capacityCeiling - math.Exp(logRemaining), true
}

// capacityTangents is pchipTangents plus a never-overquote sandwich clamp
// (instead of amountOutTangents' floor).
//
// Overquoting the real amountOut corresponds to this curve dipping BELOW its
// own chord -- the mirror image of the primary fit, because
// amountOut = capacityCeiling - exp(logRemaining) inverts direction.
// Re-deriving the same p(t)-chord(t) factorization for this segment (see
// amountOutTangents) gives the opposite requirement: p(t) >= chord(t) iff
// m_lo >= S and m_hi <= S. Unlike the primary fit's floor (only satisfiable
// on one side of a decelerating node, never both), this sandwich condition
// IS satisfiable on both sides whenever secant[i-1] >= secant[i] -- i.e.
// whenever capacity is depleting at least as fast in the segment after this
// node as before it, which is exactly the common "approaching a cliff"
// shape -- so both segments can be protected at once here. Applied after
// fcMonotonicitySafety, unlike amountOutTangents' floor, since here it's the
// clamp itself doing the monotonicity-preserving work (see the derivation).
func capacityTangents(x, y []float64) []float64 {
	tangent, secant := pchipTangents(x, y)
	if len(secant) == 0 {
		return tangent
	}

	fcMonotonicitySafety(tangent, secant)

	for i := 1; i < len(tangent)-1; i++ {
		if secant[i-1] >= secant[i] {
			tangent[i] = math.Max(secant[i], math.Min(tangent[i], secant[i-1]))
		}
	}

	return tangent
}
