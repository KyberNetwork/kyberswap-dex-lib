package ladder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteAmountOut(t *testing.T) {
	t.Parallel()

	points := []Point{
		{1000, 2000},
		{2000, 3800},
		{5000, 9000},
	}

	tests := []struct {
		name     string
		amountIn float64
		wantOut  float64
		wantErr  error
	}{
		{"zero amount", 0, 0, ErrZeroAmountIn},
		{"below first point, spline toward origin", 500, 1000, nil},
		{"exact first point", 1000, 2000, nil},
		{"between points 0 and 1", 1500, 2923.606797749979, nil},
		{"exact match at later point", 2000, 3800, nil},
		{"between points 1 and 2", 3500, 6400, nil},
		{"exceeds last point", 6000, 0, ErrAmountInTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := QuoteAmountOut(points, tt.amountIn)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.InDelta(t, tt.wantOut, out, 1e-9)
		})
	}
}

func TestQuoteAmountOut_NoPoints(t *testing.T) {
	t.Parallel()
	_, err := QuoteAmountOut(nil, 100)
	assert.ErrorIs(t, err, ErrNoQuote)
}

// TestSpline_Monotonic guards the property that actually motivates a
// monotone spline over a plain cubic spline: amountOut must never decrease
// as amountIn increases, even with unevenly-spaced, curved sample points.
func TestSpline_Monotonic(t *testing.T) {
	t.Parallel()

	points := []Point{
		{10, 15},
		{50, 70},
		{100, 130},
		{2000, 2100},
		{2100, 2101}, // near-flat segment right after a much steeper one
		{9900, 9000}, // diminishing returns near the top of the curve
	}
	spline := NewSpline(points)

	const steps = 2000
	prevOut := -1.0
	for i := 1; i <= steps; i++ {
		amountIn := float64(i) / steps * 9900
		out, err := spline.QuoteAmountOut(amountIn)
		if err != nil {
			continue
		}
		assert.GreaterOrEqualf(t, out, prevOut, "amountOut decreased at amountIn=%v", amountIn)
		prevOut = out
	}
}
