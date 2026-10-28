package ladder

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A tracker probing past a pool's fill cap gets the same quoted amountIn back
// for every probe there, which used to leave a zero-width segment at the tail.
// That segment's secant came out 0/0, the control point went NaN, and every
// quote on the pool panicked. The repeat must be dropped, and dropping it must
// not shift the curve at all.
func TestNewSpline_DropsDuplicatePoints(t *testing.T) {
	t.Parallel()

	unique := []Point{
		{100, 200},
		{400, 700},
		{546.443161, 2072.148736},
	}
	capped := append(append([]Point{}, unique...),
		Point{546.443161, 2072.148736},
		Point{546.443161, 2072.148736},
		Point{546.443161, 2072.148736},
	)

	for _, amountIn := range []float64{1, 50, 100, 250, 400, 500, 546.443161} {
		want, err := NewSpline(unique).QuoteAmountOut(amountIn)
		assert.NoError(t, err)

		got, err := NewSpline(capped).QuoteAmountOut(amountIn)
		assert.NoError(t, err)
		assert.False(t, math.IsNaN(got))
		assert.Equal(t, want, got)
	}

	// The cap is still the last quotable amount: past it the pool is unknown,
	// so the spline must refuse rather than extrapolate.
	_, err := NewSpline(capped).QuoteAmountOut(547)
	assert.ErrorIs(t, err, ErrAmountInTooLarge)
}
