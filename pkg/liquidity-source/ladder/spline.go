package ladder

import (
	"math"
	"sort"
)

// Spline is a monotone cubic Hermite interpolant (Fritsch-Carlson) through a
// set of on-chain-probed points, plus an implicit (0,0) anchor so amounts
// below the first real sample interpolate smoothly toward the origin instead
// of needing a separate pro-rated special case. Monotone Hermite is used
// instead of a plain/natural cubic spline because it cannot overshoot
// between points -- important here since a swap curve must never imply a
// negative or decreasing amountOut for an increasing amountIn, which an
// unconstrained spline can produce when points are unevenly spaced.
type Spline struct {
	x, y, tangent []float64
}

// NewSpline builds the spline from points sorted ascending by AmountIn.
func NewSpline(points []Point) *Spline {
	n := len(points)
	x := make([]float64, n+1)
	y := make([]float64, n+1)
	for i, p := range points {
		x[i+1], y[i+1] = p[0], p[1]
	}
	return &Spline{x: x, y: y, tangent: fritschCarlsonTangents(x, y)}
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

	return h00*s.y[lo] + h10*h*s.tangent[lo] + h01*s.y[hi] + h11*h*s.tangent[hi], nil
}

// fritschCarlsonTangents computes monotonicity-preserving tangents for cubic
// Hermite interpolation through (x[i], y[i]).
func fritschCarlsonTangents(x, y []float64) []float64 {
	n := len(x)
	tangent := make([]float64, n)
	if n < 2 {
		return tangent
	}

	secant := make([]float64, n-1)
	for i := range n - 1 {
		secant[i] = (y[i+1] - y[i]) / (x[i+1] - x[i])
	}

	tangent[0] = secant[0]
	tangent[n-1] = secant[n-2]
	for i := 1; i < n-1; i++ {
		tangent[i] = (secant[i-1] + secant[i]) / 2
	}

	for i := range n - 1 {
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

	return tangent
}

// QuoteAmountOut is a convenience one-shot wrapper around NewSpline, for
// callers that don't need to query the same curve repeatedly. Repeated
// callers (e.g. PoolSimulator, quoting many times per route) should build a
// *Spline once instead.
func QuoteAmountOut(points []Point, amountIn float64) (float64, error) {
	return NewSpline(points).QuoteAmountOut(amountIn)
}
