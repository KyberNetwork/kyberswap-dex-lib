package ladder

import "math"

// pchipTangents computes PCHIP (Fritsch-Butland weighted harmonic mean)
// initial tangents for a monotone cubic Hermite fit through (x[i], y[i]).
//
// It returns the tangents alongside the underlying secants rather than
// applying fcMonotonicitySafety itself: both fits in this package (see
// amountOutTangents and capacityTangents) still need to apply their own
// never-overquote clamp relative to those secants, and each needs it in a
// different order relative to fcMonotonicitySafety, so that step is left to
// the caller.
func pchipTangents(x, y []float64) (tangent, secant []float64) {
	n := len(x)
	tangent = make([]float64, n)
	if n < 2 {
		return tangent, nil
	}

	secant = make([]float64, n-1)
	for i := range n - 1 {
		secant[i] = (y[i+1] - y[i]) / (x[i+1] - x[i])
	}

	tangent[0] = secant[0]
	tangent[n-1] = secant[n-2]
	for i := 1; i < n-1; i++ {
		// Weighted harmonic mean instead of a plain arithmetic average: it
		// skews hard toward whichever neighboring secant is flatter, which
		// hugs decelerating/knee-shaped curves much more tightly than an
		// average does.
		h0, h1 := x[i]-x[i-1], x[i+1]-x[i]
		s0, s1 := secant[i-1], secant[i]
		if s0 == 0 || s1 == 0 || (s0 > 0) != (s1 > 0) {
			tangent[i] = 0
			continue
		}
		w1, w2 := 2*h1+h0, h1+2*h0
		tangent[i] = (w1 + w2) / (w1/s0 + w2/s1)
	}

	return tangent, secant
}

// fcMonotonicitySafety applies Fritsch-Carlson's monotonicity clamp to
// tangent in place: it bounds each tangent so the cubic segment it belongs
// to can't overshoot past its own endpoint values, regardless of what
// produced the tangent (PCHIP, or a caller's own never-overquote clamp)
// beforehand.
func fcMonotonicitySafety(tangent, secant []float64) {
	for i := range secant {
		m := secant[i]
		if m == 0 {
			tangent[i], tangent[i+1] = 0, 0
			continue
		}
		a, b := tangent[i]/m, tangent[i+1]/m
		if a < 0 {
			tangent[i], a = 0, 0
		}
		if b < 0 {
			tangent[i+1], b = 0, 0
		}
		if s := a*a + b*b; s > 9 {
			tau := 3 / math.Sqrt(s)
			tangent[i] = tau * a * m
			tangent[i+1] = tau * b * m
		}
	}
}
