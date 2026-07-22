package ladder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// liquidcoreUSDCkHYPELadder is a real ladder captured from the liquidcore
// USDC/kHYPE pool on hyperevm (0x158f5919a3c65c201a02cb2fee7421f7b78f3b1e),
// probed via estimateSwapBatch at block 41068712. The two test cases below
// are ground-truthed against estimateSwap calls at that same block.
var liquidcoreUSDCkHYPELadder = []Point{
	{70907263, 1127404916627193008},
	{113451621, 1803847869783450808},
	{177268158, 2818484085051797647},
	{283629053, 4509574539262786397},
	{446715759, 7102508807231598607},
	{709072634, 11273710656277336679},
	{1120334763, 17812106226417052340},
	{1772681587, 28182865949184242983},
	{2807927634, 44639425080841007054},
	{4445885420, 70674136371705474840},
	{7041091264, 111914326592201581888},
	{11153712547, 177243155629748030120},
	{17670090062, 280684939878899203883},
	{27987096900, 346848717530822469076},
	{44324130408, 346848717530822469076},
	{70198190857, 346848717530822469076},
}

// TestQuoteAmountOut_LiquidcoreOverquote guards against the spline quoting
// meaningfully more than the pool would actually pay out. We're generally ok
// underquoting (no lower bound asserted here) but overquoting risks promising
// a swap the pool can't honor.
func TestQuoteAmountOut_LiquidcoreOverquote(t *testing.T) {
	t.Parallel()

	const maxOverquotePct = 0.15

	cases := []struct {
		name     string
		amountIn float64
		actual   float64 // ground truth from estimateSwap at block 41068712
	}{
		// Sits inside a decelerating segment right before the reserve-cap
		// plateau; plain Fritsch-Carlson tangents bulge the interpolant
		// above both the linear chord and the real contract quote here.
		{"decelerating segment before the cap", 15450148532, 245484082375031380493},
		// Sits in the next segment, where the real curve has a sharp cliff
		// up to near-max output that the wide geometric sample gap misses
		// entirely. Both the chord and the spline undershoot heavily here;
		// that's fine, we only assert no overquote.
		{"missed cliff right after the same segment", 22828593481, 346869458618987283270},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			quoted, err := QuoteAmountOut(liquidcoreUSDCkHYPELadder, tc.amountIn)
			assert.NoError(t, err)

			diffPct := (quoted - tc.actual) / tc.actual * 100
			t.Logf("quoted=%.6e actual=%.6e diff=%.4f%%", quoted, tc.actual, diffPct)
			assert.LessOrEqualf(t, diffPct, maxOverquotePct,
				"overquoted by %.4f%%, want at most %.2f%%", diffPct, maxOverquotePct)
		})
	}
}

// TestQuoteAmountOutLiquidcore_Monotonic guards the capacity-space blend
// specifically: it only activates near the reserve cap, right where the
// blend weight is transitioning fastest, so this is the fixture most likely
// to expose a dip that the generic toy fixture in TestSpline_Monotonic
// wouldn't catch.
func TestQuoteAmountOutLiquidcore_Monotonic(t *testing.T) {
	t.Parallel()

	spline := NewSpline(liquidcoreUSDCkHYPELadder)
	first := liquidcoreUSDCkHYPELadder[0].AmountIn()
	last := liquidcoreUSDCkHYPELadder[len(liquidcoreUSDCkHYPELadder)-1].AmountIn()

	const steps = 200_000
	prevOut := -1.0
	for i := 1; i <= steps; i++ {
		amountIn := first + float64(i)/steps*(last-first)
		out, err := spline.QuoteAmountOut(amountIn)
		if err != nil {
			continue
		}
		assert.GreaterOrEqualf(t, out, prevOut, "amountOut decreased at amountIn=%v", amountIn)
		prevOut = out
	}
}
