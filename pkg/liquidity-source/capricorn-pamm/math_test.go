package capricornpamm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func u(v uint64) *uint256.Int { return uint256.NewInt(v) }

// uDec parses a decimal string into uint256.
func uDec(t *testing.T, s string) *uint256.Int {
	t.Helper()
	v, err := uint256.FromDecimal(s)
	require.NoError(t, err)
	return v
}

// pt builds a ladder point from two decimal strings.
func pt(t *testing.T, in, out string) LadderPoint {
	t.Helper()
	return LadderPoint{AmountIn: uDec(t, in), AmountOut: uDec(t, out)}
}

func TestQuoteAmountOut_RejectsZeroAmount(t *testing.T) {
	ladder := []LadderPoint{pt(t, "1000000", "28860549919257837724")}

	_, err := QuoteAmountOut(ladder, u(0))
	assert.ErrorIs(t, err, ErrZeroAmount, "zero amountIn must return ErrZeroAmount")
}

func TestQuoteAmountOut_RejectsNilAmount(t *testing.T) {
	ladder := []LadderPoint{pt(t, "1000000", "28860549919257837724")}

	_, err := QuoteAmountOut(ladder, nil)
	assert.ErrorIs(t, err, ErrZeroAmount, "nil amountIn must return ErrZeroAmount")
}

func TestQuoteAmountOut_RejectsEmptyLadder(t *testing.T) {
	_, err := QuoteAmountOut(nil, u(1))
	assert.ErrorIs(t, err, ErrNoQuote, "empty ladder must return ErrNoQuote")

	_, err = QuoteAmountOut([]LadderPoint{}, u(1))
	assert.ErrorIs(t, err, ErrNoQuote, "empty ladder must return ErrNoQuote")
}

func TestQuoteAmountOut_RejectsAmountAboveLargestGridPoint(t *testing.T) {
	// Grid: 1 USDC -> 28.86 WMON, 10 USDC -> 285.9 WMON (synthetic).
	ladder := []LadderPoint{
		pt(t, "1000000", "28860549919257837724"),
		pt(t, "10000000", "285900000000000000000"),
	}
	// 100 USDC > 10 USDC (largest grid) -> reject.
	_, err := QuoteAmountOut(ladder, uDec(t, "100000000"))
	assert.ErrorIs(t, err, ErrAmountInTooLarge,
		"amountIn beyond largest grid point must return ErrAmountInTooLarge")
}

func TestQuoteAmountOut_ExactGridHit_SmallestPoint(t *testing.T) {
	// Smallest grid point: amountIn == ladder[0].AmountIn -> hit the
	// scale-from-origin branch which returns out * in / in == out exactly.
	out := uDec(t, "28860549919257837724")
	in := u(1_000_000)
	ladder := []LadderPoint{
		{AmountIn: in, AmountOut: out},
	}

	got, err := QuoteAmountOut(ladder, in)
	require.NoError(t, err)
	assert.Equal(t, out.String(), got.String(),
		"exact hit on smallest grid point must return its AmountOut exactly")
}

func TestQuoteAmountOut_ExactGridHit_MiddlePoint(t *testing.T) {
	// Three-point ladder, hit the middle one exactly.
	ladder := []LadderPoint{
		pt(t, "1000000", "28860549919257837724"),
		pt(t, "10000000", "285900000000000000000"),
		pt(t, "100000000", "2700000000000000000000"),
	}
	got, err := QuoteAmountOut(ladder, u(10_000_000))
	require.NoError(t, err)
	assert.Equal(t, "285900000000000000000", got.String(),
		"exact hit on middle grid point must return its AmountOut exactly")
}

func TestQuoteAmountOut_ExactGridHit_LargestPoint(t *testing.T) {
	// Three-point ladder, hit the largest point exactly.
	ladder := []LadderPoint{
		pt(t, "1000000", "28860549919257837724"),
		pt(t, "10000000", "285900000000000000000"),
		pt(t, "100000000", "2700000000000000000000"),
	}
	got, err := QuoteAmountOut(ladder, u(100_000_000))
	require.NoError(t, err)
	assert.Equal(t, "2700000000000000000000", got.String(),
		"exact hit on largest grid point must return its AmountOut exactly")
}

func TestQuoteAmountOut_BelowSmallestGridPoint_LinearFromOrigin(t *testing.T) {
	// Grid: 1_000_000 USDC raw -> 28.86 WMON.
	// Query 500_000 (half of smallest) -> expect floor(500_000 * 28.86… / 1_000_000) = 14.43… WMON
	ladder := []LadderPoint{
		pt(t, "1000000", "28860549919257837724"),
	}

	got, err := QuoteAmountOut(ladder, u(500_000))
	require.NoError(t, err)

	// 500_000 * 28860549919257837724 / 1_000_000 = 14_430_274_959_628_918_862 (floor)
	assert.Equal(t, "14430274959628918862", got.String(),
		"below-smallest must scale linearly from the origin")
}

func TestQuoteAmountOut_BetweenGridPoints_LinearInterp(t *testing.T) {
	// Two-point ladder. Linearly interp at the midpoint.
	//   in: 1_000_000 -> out: 1_000
	//   in: 3_000_000 -> out: 2_000
	// At in=2_000_000: out = 1_000 + (2_000_000 - 1_000_000) * (2_000 - 1_000) / (3_000_000 - 1_000_000)
	//                       = 1_000 + 1_000_000 * 1_000 / 2_000_000
	//                       = 1_000 + 500
	//                       = 1_500
	ladder := []LadderPoint{
		{AmountIn: u(1_000_000), AmountOut: u(1_000)},
		{AmountIn: u(3_000_000), AmountOut: u(2_000)},
	}

	got, err := QuoteAmountOut(ladder, u(2_000_000))
	require.NoError(t, err)
	assert.Equal(t, uint64(1_500), got.Uint64())
}

func TestQuoteAmountOut_BetweenGridPoints_TruncatesToward_Floor(t *testing.T) {
	// Same shape but with a midpoint that doesn't divide evenly:
	//   in: 100 -> out: 0
	//   in: 300 -> out: 1
	// At in=200: out = 0 + (200 - 100) * (1 - 0) / (300 - 100) = 100 / 200 = 0 (floor).
	ladder := []LadderPoint{
		{AmountIn: u(100), AmountOut: u(0)},
		{AmountIn: u(300), AmountOut: u(1)},
	}
	got, err := QuoteAmountOut(ladder, u(200))
	require.NoError(t, err)
	assert.Equal(t, uint64(0), got.Uint64(),
		"interpolation must floor (truncate toward zero), matching on-chain uint math")
}

func TestQuoteAmountOut_BetweenGridPoints_ThreePointLadder(t *testing.T) {
	// Three-point ladder, query inside the second segment.
	//   in: 1_000     -> out: 100
	//   in: 10_000    -> out: 950
	//   in: 100_000   -> out: 8_000
	// At in=50_000 (halfway between 10_000 and 100_000):
	//   out = 950 + (50_000 - 10_000) * (8_000 - 950) / (100_000 - 10_000)
	//       = 950 + 40_000 * 7_050 / 90_000
	//       = 950 + 282_000_000 / 90_000
	//       = 950 + 3_133  (floor of 3133.33…)
	//       = 4_083
	ladder := []LadderPoint{
		{AmountIn: u(1_000), AmountOut: u(100)},
		{AmountIn: u(10_000), AmountOut: u(950)},
		{AmountIn: u(100_000), AmountOut: u(8_000)},
	}
	got, err := QuoteAmountOut(ladder, u(50_000))
	require.NoError(t, err)
	assert.Equal(t, uint64(4_083), got.Uint64())
}

func TestQuoteAmountOut_OnChainSmokeVector(t *testing.T) {
	// Captured from dex-explorer smoke test against USDC/WMON pool
	// at Tenderly fork of Monad mainnet block 73_874_829:
	//   pool.quoteExactIn(USDC, 1e6) == 28_903_623_346_225_898_755 (= 28.903 WMON)
	// Single-point ladder; query at the exact grid point must return the exact value.
	ladder := []LadderPoint{
		pt(t, "1000000", "28903623346225898755"),
	}
	got, err := QuoteAmountOut(ladder, u(1_000_000))
	require.NoError(t, err)
	assert.Equal(t, "28903623346225898755", got.String(),
		"smoke vector: 1 USDC -> 28.9036... WMON must be returned exactly")
}

func TestQuoteAmountOut_LinearMonotonicLadder_DoesNotPanic(t *testing.T) {
	// A degenerate but valid ladder: dy/dx = constant. Interp should equal
	// the trivial scaling at every point.
	//   in: 1 -> out: 5
	//   in: 100 -> out: 500
	// At in=50: 5 + (50 - 1) * (500 - 5) / (100 - 1) = 5 + 49 * 495 / 99 = 5 + 245 = 250.
	ladder := []LadderPoint{
		{AmountIn: u(1), AmountOut: u(5)},
		{AmountIn: u(100), AmountOut: u(500)},
	}
	got, err := QuoteAmountOut(ladder, u(50))
	require.NoError(t, err)
	assert.Equal(t, uint64(250), got.Uint64())
}

func TestQuoteAmountOut_BelowSmallest_TwoPointLadder_DoesNotConfuseIndex(t *testing.T) {
	// Make sure the < first.AmountIn branch fires and we don't accidentally
	// fall into the loop with a stale index.
	ladder := []LadderPoint{
		{AmountIn: u(100), AmountOut: u(50)},
		{AmountIn: u(1000), AmountOut: u(400)},
	}
	got, err := QuoteAmountOut(ladder, u(10))
	require.NoError(t, err)
	// 10 * 50 / 100 = 5
	assert.Equal(t, uint64(5), got.Uint64())
}

func TestQuoteAmountOut_BelowSmallest_BoundaryAtExactlyFirstAmountIn(t *testing.T) {
	// At exactly amountIn == first.AmountIn the cheap-path returns
	// `amountIn * first.AmountOut / first.AmountIn` == first.AmountOut.
	// This serves two purposes:
	//   1. Pins the boundary semantics of the `<=` comparison.
	//   2. Documents (in code) that the below-smallest branch is
	//      mathematically incapable of overflowing the 256-bit final
	//      result — see math.go for the proof.
	maxU := new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 256), uint256.NewInt(1))
	ladder := []LadderPoint{
		{AmountIn: new(uint256.Int).Lsh(uint256.NewInt(1), 100), AmountOut: maxU},
	}
	got, err := QuoteAmountOut(ladder, new(uint256.Int).Lsh(uint256.NewInt(1), 100))
	require.NoError(t, err)
	assert.Equal(t, maxU.String(), got.String(),
		"amountIn == first.AmountIn must return first.AmountOut exactly, even at the uint256 ceiling")
}
