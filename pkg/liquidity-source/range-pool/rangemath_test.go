package rangepool

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/math"
)

// This is the Go port of ParaSwap's src/dex/range-pool/lib/rangeMath.test.ts. It proves
// that CalcOutGivenIn / CalcInGivenOut reproduce the weighted-product curve bit-for-bit,
// cross-checked against the validated balancer/v3/math.WeightedMath for sub-30% amounts
// (where WeightedMath's MAX_IN/OUT_RATIO guard does not fire — RangeMath has no such
// guard, reference §3), plus the fact-balance cap and the EXACT_OUT revert guards. Pure
// offline unit tests, no RPC.

// wad = 1e18.
var wad = uint256.NewInt(1e18)

// mulU returns n * wad as a fresh uint256.
func mulWad(n uint64) *uint256.Int {
	return new(uint256.Int).Mul(uint256.NewInt(n), wad)
}

// weight returns a normalized weight from a percentage-in-hundredths, i.e. weightPct(50)
// = 0.5e18. Matches the TS `50n * 10n ** 16n` form.
func weightPct(pct uint64) *uint256.Int {
	return new(uint256.Int).Mul(uint256.NewInt(pct), uint256.NewInt(1e16))
}

// lcg is the deterministic LCG from the TS test, so the cross-check is reproducible.
func lcg(seed uint32) func() float64 {
	s := seed
	return func() float64 {
		s = 1664525*s + 1013904223 // uint32 arithmetic wraps, matching >>> 0
		return float64(s) / 4294967296.0
	}
}

// Weight pairs (scaled18): the y==1 shortcut (50/50), the y==4 region (80/20), y==0.25
// (20/80), and non-shortcut ratios (7/20, 20/7, 10/7) that exercise LogExpMath.
func weightPairs() [][2]*uint256.Int {
	return [][2]*uint256.Int{
		{weightPct(50), weightPct(50)},
		{weightPct(80), weightPct(20)},
		{weightPct(20), weightPct(80)},
		{weightPct(7), weightPct(20)},
		{weightPct(20), weightPct(7)},
		{weightPct(10), weightPct(7)},
	}
}

func balances() []*uint256.Int {
	return []*uint256.Int{
		mulWad(1_000),
		mulWad(30_000),
		mulWad(1_234_567),
		mulWad(5),
		mulWad(999_999_999),
	}
}

// hugeFact is large enough that the fact cap never binds (isolates the curve math).
var hugeFact = uint256.MustFromDecimal("10000000000000000000000000000000000000000") // 1e40

const oneMillion = 1_000_000

func TestRangeMath_CalcOutGivenIn_MatchesWeightedMath(t *testing.T) {
	rand := lcg(0xc0ffee)
	checks := 0
	for _, wp := range weightPairs() {
		wIn, wOut := wp[0], wp[1]
		for _, vIn := range balances() {
			for _, vOut := range balances() {
				for i := 0; i < 8; i++ {
					// amountIn strictly under the 30% MAX_IN_RATIO so WeightedMath
					// doesn't throw: frac up to ~29% of vIn.
					frac := uint256.NewInt(uint64(rand() * 29_0000))
					amountIn := new(uint256.Int).Div(
						new(uint256.Int).Mul(vIn, frac),
						uint256.NewInt(100_0000),
					)
					if amountIn.IsZero() {
						continue
					}

					mine, err := CalcOutGivenIn(vIn, wIn, vOut, wOut, amountIn, hugeFact)
					require.NoError(t, err)

					ref, err := math.WeightedMath.ComputeOutGivenExactIn(vIn, wIn, vOut, wOut, amountIn)
					require.NoError(t, err)

					assert.Equal(t, ref, mine,
						"calcOutGivenIn mismatch: wIn=%s wOut=%s vIn=%s vOut=%s amountIn=%s",
						wIn, wOut, vIn, vOut, amountIn)
					checks++
				}
			}
		}
	}
	assert.Greater(t, checks, 500)
}

func TestRangeMath_CalcInGivenOut_MatchesWeightedMath(t *testing.T) {
	rand := lcg(0xbeef)
	checks := 0
	for _, wp := range weightPairs() {
		wIn, wOut := wp[0], wp[1]
		for _, vIn := range balances() {
			for _, vOut := range balances() {
				for i := 0; i < 8; i++ {
					// amountOut under 30% of vOut so WeightedMath doesn't throw.
					frac := uint256.NewInt(uint64(rand() * 29_0000))
					amountOut := new(uint256.Int).Div(
						new(uint256.Int).Mul(vOut, frac),
						uint256.NewInt(100_0000),
					)
					if amountOut.IsZero() {
						continue
					}

					mine, err := CalcInGivenOut(vIn, wIn, vOut, wOut, amountOut)
					require.NoError(t, err)

					ref, err := math.WeightedMath.ComputeInGivenExactOut(vIn, wIn, vOut, wOut, amountOut)
					require.NoError(t, err)

					assert.Equal(t, ref, mine,
						"calcInGivenOut mismatch: wIn=%s wOut=%s vIn=%s vOut=%s amountOut=%s",
						wIn, wOut, vIn, vOut, amountOut)
					checks++
				}
			}
		}
	}
	assert.Greater(t, checks, 500)
}

// TestRangeMath_CalcInGivenOut_ZeroShortCircuit asserts the amountOut==0 -> 0 guard that
// RangeMath adds (avoiding a dust charge from powUp rounding on base==1e18).
func TestRangeMath_CalcInGivenOut_ZeroShortCircuit(t *testing.T) {
	// Use a non-shortcut exponent (10/7) where powUp(1e18, y) would otherwise round up
	// above 1e18 and produce a non-zero input for zero output.
	out, err := CalcInGivenOut(mulWad(1_000), weightPct(10), mulWad(1_000), weightPct(7), uint256.NewInt(0))
	require.NoError(t, err)
	assert.True(t, out.IsZero())
}

// TestRangeMath_CapsOutputAtFactBalance proves CalcOutGivenIn caps at factBalanceOut-1e6
// AND has no 30% guard: amountIn here is 1/3 of vIn (>30%), which WeightedMath would
// reject, but RangeMath computes the curve then caps.
func TestRangeMath_CapsOutputAtFactBalance(t *testing.T) {
	w := weightPct(50)
	vIn := mulWad(30_000)
	vOut := mulWad(30_000)
	amountIn := mulWad(10_000) // 33% of vIn -> would trip WeightedMath's MAX_IN_RATIO
	smallFact := mulWad(5)     // far below the uncapped curve output (~7500e18)

	capped, err := CalcOutGivenIn(vIn, w, vOut, w, amountIn, smallFact)
	require.NoError(t, err)

	expected := new(uint256.Int).Sub(smallFact, AbsoluteMinTokenBalance)
	assert.Equal(t, expected, capped)

	// Sanity: WeightedMath indeed refuses this input (confirms the guard divergence).
	_, refErr := math.WeightedMath.ComputeOutGivenExactIn(vIn, w, vOut, w, amountIn)
	assert.ErrorIs(t, refErr, math.ErrMaxInRatio)
}

func TestRangeMath_CapsOutputAtZeroFact(t *testing.T) {
	w := weightPct(50)
	v := mulWad(30_000)
	// factBalanceOut below AbsoluteMinTokenBalance -> maxOut = 0.
	out, err := CalcOutGivenIn(v, w, v, w, mulWad(1_000), uint256.NewInt(oneMillion-1))
	require.NoError(t, err)
	assert.True(t, out.IsZero())
}

func TestRangeMath_IsExactOutAllowed(t *testing.T) {
	// allowed: amountOut + 1e6 <= factOut and amountOut < vOut.
	assert.True(t, IsExactOutAllowed(mulWad(10), mulWad(100), mulWad(1_000)))
	// amountOut + 1e6 > factBalanceOut.
	assert.False(t, IsExactOutAllowed(mulWad(100), mulWad(100), mulWad(1_000)))
	// amountOut >= virtualBalanceOut.
	assert.False(t, IsExactOutAllowed(mulWad(1_000), mulWad(5_000), mulWad(1_000)))
}

func TestRangeMath_CalcSpotPrice(t *testing.T) {
	// base/quote = 30000 with equal weights -> 30000e18.
	p, err := CalcSpotPrice(mulWad(30_000), weightPct(50), wad, weightPct(50))
	require.NoError(t, err)
	assert.Equal(t, mulWad(30_000), p)
}

// TestRangeMath_OversizedExactIn_OverflowIsError is the cross-check for bugbot finding #4
// (adapters/docs/bugbot-findings-crosscheck.md), the pure-math one that applies in every
// language. The weighted-product curve runs LogExpMath.pow BEFORE the fact-balance cap. On
// a NON-short-circuit weight ratio (wIn/wOut not 1/2/4) with wIn > wOut (exponent > 1e18),
// a large enough exact-in drives the curve base toward 0, so pow exceeds its product bound.
// CalcOutGivenIn must then return an ERROR (matching the on-chain querySwap revert for that
// size) — never a garbage value and never a panic — and, being a pure function, one
// erroring amount must not affect the next (per-amount failure isolation, which Kyber gets
// by construction since CalcAmountOut is called per amount).
func TestRangeMath_OversizedExactIn_OverflowIsError(t *testing.T) {
	// 20/6 = 3.333e18 — the same non-short-circuit, wIn>wOut ratio as the live 8-token
	// pool's WETH(0.20)->SHIB(0.06) direction; 50/50 and 80/20 would short-circuit and
	// could not reproduce the overflow.
	wIn, wOut := weightPct(20), weightPct(6)
	vIn, vOut := mulWad(1_000), mulWad(1_000)
	factOut := mulWad(1_000)

	// An exact-in ~1e7x the virtual balance underflows the base past LogExpMath's
	// MIN_NATURAL_EXPONENT bound. Must not panic; must be a clean error.
	oversized := new(uint256.Int).Mul(vIn, uint256.NewInt(10_000_000))
	require.NotPanics(t, func() {
		out, err := CalcOutGivenIn(vIn, wIn, vOut, wOut, oversized, factOut)
		require.Error(t, err, "oversized exact-in must be no-price (pow overflow), not a value")
		assert.Nil(t, out)
	})

	// Isolation: a normal amount right after the erroring one still prices correctly —
	// CalcOutGivenIn is pure, so one bad amount cannot poison the next.
	out, err := CalcOutGivenIn(vIn, wIn, vOut, wOut, mulWad(10), factOut)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.IsZero())
}
