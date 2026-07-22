package ladder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// liquidcoreUSDCkHYPELadder is a real ladder captured from the liquidcore
// USDC/kHYPE pool on hyperevm (0x158f5919a3c65c201a02cb2fee7421f7b78f3b1e),
// block 41125109. Unlike earlier captures, the ladder points AND the test
// cases' ground truth below all came from a single estimateSwapBatch call
// (grid points and test amounts probed together) rather than a separate
// follow-up call: estimateSwap on this contract doesn't actually respect
// historical block pinning the way getReserves does (confirmed by re-issuing
// an identical pinned-block call seconds apart and getting different
// results), so any ground truth fetched as a second, later call is not
// reliably describing the same state as the ladder it's being compared
// against. Fetching everything in one batch sidesteps that entirely.
var liquidcoreUSDCkHYPELadder = []Point{
	{44629725, 747540412135206769},
	{71407560, 1196064659416330830},
	{111574313, 1868851038712933103},
	{178518901, 2990131734341676528},
	{281167270, 4709410355836777207},
	{446297254, 7475179706233270832},
	{705149661, 11810665703239956313},
	{1115743135, 18687200991856089394},
	{1767337127, 29599341124944004872},
	{2798283784, 46862339267459824931},
	{4431731735, 74209215598361446501},
	{7020255811, 117530433282122961696},
	{11121727579, 186130437886950183508},
	{17615352630, 259983194262058607184},
	{27898041371, 259983194262058607184},
	{44183428183, 259983194262058607184},
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
		actual   float64 // ground truth from the same atomic estimateSwapBatch call as the ladder
	}{
		{"inside the knee", 12_000_000_000, 200812888803358591038},
		{"further into the knee", 14_000_000_000, 234237105343538890740},
		{"just before the cap", 16_000_000_000, 259983194262058607184},
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

// liquidcoreUSDCkHYPELadder2 is a second, independent atomic capture (same
// pool, a later block) whose knee exposed a real overquote under this
// package's previous PCHIP-based spline (its never-overquote floor and
// capacity-space blend were each proven safe only under their own specific
// conditions, and this ladder's knee fell outside both). The current
// Spline (a clamped rational quadratic Bezier per segment, see spline.go)
// quotes safely here -- confirmed the same way as liquidcoreUSDCkHYPELadder,
// with the ladder points and test amounts all probed in one
// estimateSwapBatch call.
var liquidcoreUSDCkHYPELadder2 = []Point{
	{44635176, 749780179858591812},
	{71416282, 1199648294492933399},
	{111587940, 1874450449646479532},
	{178540705, 2999090716806711089},
	{281201610, 4723567872671395675},
	{446351763, 7497651751851536687},
	{705235785, 11846052605392206686},
	{1115879408, 18743378902385919530},
	{1767552982, 29688026215240113178},
	{2798625555, 47003218183278916403},
	{4432273009, 74432305898067823626},
	{7021113236, 117883758483911667373},
	{11123085940, 186688124519608278812},
	{17617504096, 259905076443769693639},
	{27901448722, 259905076443769693639},
	{44188824564, 259905076443769693639},
}

// TestQuoteAmountOut_LiquidcoreOverquote2 documents a real overquote this
// package's previous PCHIP-based spline had on liquidcoreUSDCkHYPELadder2's
// knee (at 15e9, PCHIP quoted ~0.6% above the atomically-verified ground
// truth). The current Spline (see spline.go) doesn't overquote here.
func TestQuoteAmountOut_LiquidcoreOverquote2(t *testing.T) {
	t.Parallel()

	const maxOverquotePct = 0.15

	cases := []struct {
		name     string
		amountIn float64
		actual   float64 // ground truth from the same atomic estimateSwapBatch call as the ladder
	}{
		{"inside the knee", 13_000_000_000, 218152796389921912980},
		{"further into the knee", 15_000_000_000, 251664323751719985741},
		{"just before the cap", 16_500_000_000, 259905076443769693639},
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

// TestQuoteAmountOutLiquidcore_Monotonic guards the segments right at the
// reserve cap specifically, where the flat plateau meets the last curved
// segment -- this is the fixture most likely to expose a dip that the
// generic toy fixture in TestSpline_Monotonic wouldn't catch.
func TestQuoteAmountOutLiquidcore_Monotonic(t *testing.T) {
	t.Parallel()

	for _, ladder := range [][]Point{liquidcoreUSDCkHYPELadder, liquidcoreUSDCkHYPELadder2} {
		spline := NewSpline(ladder)
		first := ladder[0].AmountIn()
		last := ladder[len(ladder)-1].AmountIn()

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
}
