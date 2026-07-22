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

// liquidcoreUSDCkHYPELadder2 is a second, independent capture of the same
// pool (block 41114514), taken after a large live price/liquidity move --
// reserve0 dropped from ~70.9e9 to ~44.7e9. The knee sits earlier and
// sharper than in liquidcoreUSDCkHYPELadder, and it's what first exposed
// PCHIP+floor overquoting past maxOverquotePct on real data.
var liquidcoreUSDCkHYPELadder2 = []Point{
	{44732122, 726568242150696840},
	{71571396, 1162509200435235673},
	{111830306, 1816420621619393011},
	{178928490, 2906243911595745345},
	{281812372, 4577334164823921003},
	{447321225, 7265537055258503769},
	{706767536, 11479318748439919945},
	{1118303064, 18163115425200176674},
	{1771392054, 28769222899753118469},
	{2804704085, 45548077738602994157},
	{4441899772, 72128007271728878245},
	{7036362881, 114235420577088666913},
	{11147244947, 180912201711786622668},
	{17655768782, 258502120883988381175},
	{27962049825, 258502120883988381175},
	{44284801355, 258502120883988381175},
}

// TestQuoteAmountOut_LiquidcoreOverquote2 documents overquote violations
// found on liquidcoreUSDCkHYPELadder2's knee (11.15e9-17.66e9) that
// TestQuoteAmountOut_LiquidcoreOverquote's fixture doesn't cover -- PCHIP's
// never-overquote floor is only proven safe for the segment right before a
// decelerating node, but the actual overquote here comes from the
// capacity-space fit itself: even a plain (unshaped) chord in log-log-
// remaining-capacity space overquotes by a comparable amount, since the true
// curve isn't log-linear across this particular (too-wide) sample gap.
func TestQuoteAmountOut_LiquidcoreOverquote2(t *testing.T) {
	t.Parallel()

	const maxOverquotePct = 0.15

	cases := []struct {
		name     string
		amountIn float64
		actual   float64 // ground truth from estimateSwap at block 41114514
	}{
		{"inside the knee", 13_000_000_000, 210761270356908427569},
		{"right before the cap, at cheb16's real sample point", 15_329_698_408, 248699998915193755779},
		{"just before the cap", 16_000_000_000, 258522846442632530933},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			quoted, err := QuoteAmountOut(liquidcoreUSDCkHYPELadder2, tc.amountIn)
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
