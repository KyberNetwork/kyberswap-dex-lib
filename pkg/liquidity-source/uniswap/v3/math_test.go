package uniswapv3

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// smallPool is a minimal three-tick pool used across several tests.
// Tick spacing 60, fee 3000, current tick -59315, matching the poolEncoded fixture.
func makeSmallPool(t *testing.T) *Pool {
	t.Helper()
	ticks := []TickU256{
		{Index: -887220, LiquidityGross: uint256.MustFromDecimal("3191465872325806144123"), LiquidityNet: int256.MustFromDec("3191465872325806144123")},
		{Index: -79320, LiquidityGross: uint256.MustFromDecimal("59713631504779700614879"), LiquidityNet: int256.MustFromDec("59713631504779700614879")},
		{Index: 887220, LiquidityGross: uint256.MustFromDecimal("62905097377105506759002"), LiquidityNet: int256.MustFromDec("-62905097377105506759002")},
	}
	sqrtPrice := uint256.MustFromDecimal("4082682361430349352208957440")
	liquidity := uint256.MustFromDecimal("461286494113032089234462")

	pool, err := NewPool(FeeAmount(3000), *sqrtPrice, *liquidity, -59315, ticks, 60)
	require.NoError(t, err)
	return pool
}

// ---------- validateList ----------

func TestValidateList(t *testing.T) {
	t.Parallel()

	liq := uint256.NewInt(1000)
	net := int256.NewInt(1000)
	netNeg := int256.MustFromDec("-1000")

	t.Run("valid", func(t *testing.T) {
		ticks := []TickU256{
			{-60, liq, net},
			{60, liq, netNeg},
		}
		err := validateList(ticks, 60)
		require.NoError(t, err)
	})

	t.Run("zero tick spacing", func(t *testing.T) {
		err := validateList(nil, 0)
		require.ErrorIs(t, err, ErrZeroTickSpacing)
	})

	t.Run("bad spacing alignment", func(t *testing.T) {
		ticks := []TickU256{
			{-61, liq, net}, // not a multiple of 60
			{61, liq, netNeg},
		}
		err := validateList(ticks, 60)
		require.ErrorIs(t, err, ErrInvalidTickSpacing)
	})

	t.Run("non-zero net sum", func(t *testing.T) {
		ticks := []TickU256{
			{-60, liq, net},
			{60, liq, net}, // net should be negative to sum to zero
		}
		err := validateList(ticks, 60)
		require.ErrorIs(t, err, ErrZeroNet)
	})

	t.Run("unsorted", func(t *testing.T) {
		ticks := []TickU256{
			{60, liq, net},
			{-60, liq, netNeg},
		}
		err := validateList(ticks, 60)
		require.ErrorIs(t, err, ErrSorted)
	})
}

// ---------- binarySearch / nextInitializedTickIndex ----------

func TestNextInitializedTickIndex(t *testing.T) {
	t.Parallel()

	liq := uint256.NewInt(500)
	ticks := []TickU256{
		{-120, liq, int256.MustFromDec("500")},
		{-60, liq, int256.MustFromDec("-500")},
		{60, liq, int256.MustFromDec("500")},
		{120, liq, int256.MustFromDec("-500")},
	}

	// lte=true: returns the largest initialized tick ≤ tick
	idx, init, err := nextInitializedTickIndex(ticks, 0, true)
	require.NoError(t, err)
	require.Equal(t, -60, idx)
	require.True(t, init)

	// exact match
	idx, init, err = nextInitializedTickIndex(ticks, 60, true)
	require.NoError(t, err)
	require.Equal(t, 60, idx)
	require.True(t, init)

	// lte=false: returns the smallest initialized tick > tick
	idx, init, err = nextInitializedTickIndex(ticks, 0, false)
	require.NoError(t, err)
	require.Equal(t, 60, idx)
	require.True(t, init)

	// below smallest → error
	_, _, err = nextInitializedTickIndex(ticks, -200, true)
	require.ErrorIs(t, err, ErrBelowSmallest)

	// at or above largest (lte=false) → error
	_, _, err = nextInitializedTickIndex(ticks, 120, false)
	require.ErrorIs(t, err, ErrAtOrAboveLargest)
}

// ---------- NewPool ----------

func TestNewPool(t *testing.T) {
	t.Parallel()

	ticks := []TickU256{
		{-60, uint256.NewInt(100), int256.NewInt(100)},
		{60, uint256.NewInt(100), int256.MustFromDec("-100")},
	}

	// valid pool at tick 0: sqrtPrice must be between sqrtRatioAtTick(0) and sqrtRatioAtTick(1)
	// sqrtRatioAtTick(0) = 79228162514264337593543950336 (Q96)
	sqrtPrice := *uint256.MustFromDecimal("79228162514264337593543950336")
	liquidity := *uint256.NewInt(1e9)

	p, err := NewPool(FeeMedium, sqrtPrice, liquidity, 0, ticks, 60)
	require.NoError(t, err)
	require.Equal(t, 0, p.TickCurrent)
	require.Equal(t, FeeMedium, p.Fee)

	// fee too high
	_, err = NewPool(FeeMax, sqrtPrice, liquidity, 0, ticks, 60)
	require.ErrorIs(t, err, ErrFeeTooHigh)

	// sqrtPrice out of range for tick
	bad := *uint256.MustFromDecimal("1")
	_, err = NewPool(FeeMedium, bad, liquidity, 0, ticks, 60)
	require.ErrorIs(t, err, ErrInvalidSqrtRatioX96)
}

// ---------- GetOutputAmountV2 (exact input) ----------

func TestGetOutputAmountV2(t *testing.T) {
	t.Parallel()
	p := makeSmallPool(t)

	// Swap 1 WETH (1e18) in as token1 (zeroForOne=false, token1→token0)
	var amountIn uint256.Int
	amountIn.SetFromBig(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	// price limit: slightly above current sqrt price (going up, token1→token0)
	priceLimit, _ := uint256.FromDecimal("1461446703485210103287273052203988822378723970341") // MaxSqrtRatio - 1

	res, err := p.GetOutputAmountV2(false, amountIn, *priceLimit)
	require.NoError(t, err)
	require.True(t, res.AmountCalculated.Sign() > 0, "output must be positive")
	require.False(t, res.SqrtRatioX96.IsZero(), "sqrt ratio must be non-zero")
	require.False(t, res.Liquidity.IsZero(), "liquidity must be non-zero")
}

// ---------- GetInputAmountV2 (exact output) ----------

func TestGetInputAmountV2(t *testing.T) {
	t.Parallel()
	p := makeSmallPool(t)

	// We want exactly 1e15 of token0 out (zeroForOne=true, token0 output)
	var amountOut uint256.Int
	amountOut.SetFromBig(new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil))

	priceLimit, _ := uint256.FromDecimal("4295128740") // MinSqrtRatio + 1

	sr, err := p.GetInputAmountV2(true, amountOut, *priceLimit)
	require.NoError(t, err)
	require.True(t, sr.AmountCalculated.Sign() > 0, "input amount must be positive")
	require.False(t, sr.SqrtRatioX96.IsZero(), "sqrt ratio must be non-zero")

	// Cross-check: feeding the computed amountIn back into GetOutputAmountV2 should yield
	// approximately amountOut. Per-tick rounding in exactIn can reduce the output by up to
	// ticksCrossed-1 units, so gotOut may be slightly less than requested.
	priceLimit2, _ := uint256.FromDecimal("4295128740")
	res, err := p.GetOutputAmountV2(true, sr.AmountCalculated, *priceLimit2)
	require.NoError(t, err)
	gotOut := res.AmountCalculated.ToBig()
	requestedBI := amountOut.ToBig()
	delta := new(big.Int).Sub(requestedBI, gotOut) // positive if gotOut < requested
	require.True(t, delta.Sign() >= 0 && delta.Cmp(new(big.Int).Rsh(requestedBI, 10)) <= 0,
		"round-trip: out(%s) too far from requested(%s)", gotOut, requestedBI)
}

// ---------- GetTickAtSqrtRatio ----------

func TestGetTickAtSqrtRatio(t *testing.T) {
	t.Parallel()

	// ── error cases ──────────────────────────────────────────────────────────

	t.Run("error: below min sqrt ratio", func(t *testing.T) {
		var below uint256.Int
		below.SubUint64(MinSqrtRatioU256, 1)
		_, err := GetTickAtSqrtRatio(&below)
		require.ErrorIs(t, err, errInvalidSqrtRatio)
	})

	t.Run("error: at max sqrt ratio", func(t *testing.T) {
		_, err := GetTickAtSqrtRatio(MaxSqrtRatioU256)
		require.ErrorIs(t, err, errInvalidSqrtRatio)
	})

	t.Run("error: zero", func(t *testing.T) {
		_, err := GetTickAtSqrtRatio(new(uint256.Int))
		require.ErrorIs(t, err, errInvalidSqrtRatio)
	})

	// ── known absolute values ─────────────────────────────────────────────────

	t.Run("min sqrt ratio → MinTick", func(t *testing.T) {
		tick, err := GetTickAtSqrtRatio(MinSqrtRatioU256)
		require.NoError(t, err)
		require.Equal(t, MinTick, tick)
	})

	t.Run("max valid sqrt ratio → MaxTick-1", func(t *testing.T) {
		var maxValid uint256.Int
		maxValid.SubUint64(MaxSqrtRatioU256, 1)
		tick, err := GetTickAtSqrtRatio(&maxValid)
		require.NoError(t, err)
		require.Equal(t, MaxTick-1, tick)
	})

	t.Run("tick 0 sqrt price", func(t *testing.T) {
		// sqrtRatioAtTick(0) = 2^96 exactly
		sqrtP, _ := uint256.FromDecimal("79228162514264337593543950336")
		tick, err := GetTickAtSqrtRatio(sqrtP)
		require.NoError(t, err)
		require.Equal(t, 0, tick)
	})

	// ── round-trip: GetTickAtSqrtRatio(GetSqrtRatioAtTick(t)) == t ───────────
	// This exercises all three code paths (no correction, over-estimate, under-estimate).

	t.Run("round-trip dense [-1000, 1000]", func(t *testing.T) {
		for tick := -1000; tick <= 1000; tick++ {
			var sqrtP uint256.Int
			require.NoError(t, GetSqrtRatioAtTick(tick, &sqrtP))
			got, err := GetTickAtSqrtRatio(&sqrtP)
			require.NoError(t, err)
			require.Equal(t, tick, got, "tick=%d", tick)
		}
	})

	t.Run("round-trip near MinTick", func(t *testing.T) {
		var sqrtP uint256.Int
		require.NoError(t, GetSqrtRatioAtTick(MinTick, &sqrtP))
		require.Equal(t, MinSqrtRatioU256, &sqrtP)
		for tick := MinTick; tick <= MinTick+100; tick++ {
			require.NoError(t, GetSqrtRatioAtTick(tick, &sqrtP))
			got, err := GetTickAtSqrtRatio(&sqrtP)
			require.NoError(t, err)
			require.Equal(t, tick, got, "tick=%d", tick)
		}
	})

	t.Run("round-trip near MaxTick", func(t *testing.T) {
		var sqrtP uint256.Int
		require.NoError(t, GetSqrtRatioAtTick(MaxTick, &sqrtP))
		require.Equal(t, MaxSqrtRatioU256P1, &sqrtP)
		for tick := MaxTick - 101; tick < MaxTick; tick++ {
			require.NoError(t, GetSqrtRatioAtTick(tick, &sqrtP))
			got, err := GetTickAtSqrtRatio(&sqrtP)
			require.NoError(t, err)
			require.Equal(t, tick, got, "tick=%d", tick)
		}
	})

	t.Run("round-trip sparse full range", func(t *testing.T) {
		for tick := MinTick; tick < MaxTick; tick += 1000 {
			var sqrtP uint256.Int
			require.NoError(t, GetSqrtRatioAtTick(tick, &sqrtP))
			got, err := GetTickAtSqrtRatio(&sqrtP)
			require.NoError(t, err)
			require.Equal(t, tick, got, "tick=%d", tick)
		}
	})

	// ── intermediate values: sqrtP strictly between consecutive ticks ─────────
	// For tick t, if sqrtRatioAtTick(t)+1 < sqrtRatioAtTick(t+1), then
	// sqrtRatioAtTick(t)+1 must also map to t.

	t.Run("intermediate values", func(t *testing.T) {
		testTicks := []int{MinTick, -200000, -1000, -1, 0, 1, 1000, 200000, MaxTick - 2}
		for _, tick := range testTicks {
			var sqrtLo, sqrtHi uint256.Int
			require.NoError(t, GetSqrtRatioAtTick(tick, &sqrtLo))
			require.NoError(t, GetSqrtRatioAtTick(tick+1, &sqrtHi))
			var mid uint256.Int
			mid.AddUint64(&sqrtLo, 1)
			if !mid.Lt(&sqrtHi) {
				// consecutive ticks share only one sqrtP value; nothing to test
				continue
			}
			got, err := GetTickAtSqrtRatio(&mid)
			require.NoError(t, err)
			require.Equal(t, tick, got, "tick=%d mid=%s", tick, mid.Dec())
		}
	})
}

// ---------- getTick ----------

func TestGetTick(t *testing.T) {
	t.Parallel()

	liq := uint256.NewInt(1)
	ticks := []TickU256{
		{-60, liq, int256.NewInt(1)},
		{0, liq, int256.MustFromDec("-1")},
	}

	tick, err := getTick(ticks, -60)
	require.NoError(t, err)
	require.Equal(t, -60, tick.Index)

	tick, err = getTick(ticks, 0)
	require.NoError(t, err)
	require.Equal(t, 0, tick.Index)

	_, err = getTick(ticks, 99)
	require.Error(t, err)
}

// ---------- Swap with empty tick list (AllowEmptyTicks pools) ----------

func TestSwapEmptyTicks(t *testing.T) {
	t.Parallel()

	// Pool at tick 0, sqrtPrice = sqrtRatioAtTick(0), no initialized ticks.
	// This models a Uniswap V4 hook pool that manages liquidity outside the
	// standard tick bitmap.
	sqrtPrice := *uint256.MustFromDecimal("79228162514264337593543950336") // sqrtRatioAtTick(0)
	liquidity := *uint256.MustFromDecimal("1000000000000000000")           // 1e18
	p, err := NewPool(FeeMedium, sqrtPrice, liquidity, 0, nil, 60)
	require.NoError(t, err)

	amountIn := *uint256.MustFromDecimal("1000000000000") // 1e12

	t.Run("exactInput zeroForOne", func(t *testing.T) {
		res, err := p.GetOutputAmountV2(true, amountIn, uint256.Int{})
		require.NoError(t, err)
		require.True(t, res.AmountCalculated.IsZero(), "no output for empty-tick pool")
		require.Equal(t, amountIn, res.RemainingAmountIn, "full input returned as remaining")
		require.Equal(t, sqrtPrice, res.SqrtRatioX96, "price unchanged")
	})

	t.Run("exactInput !zeroForOne", func(t *testing.T) {
		res, err := p.GetOutputAmountV2(false, amountIn, uint256.Int{})
		require.NoError(t, err)
		require.True(t, res.AmountCalculated.IsZero(), "no output for empty-tick pool")
		require.Equal(t, amountIn, res.RemainingAmountIn, "full input returned as remaining")
		require.Equal(t, sqrtPrice, res.SqrtRatioX96, "price unchanged")
	})

	t.Run("exactOutput zeroForOne", func(t *testing.T) {
		amountOut := *uint256.MustFromDecimal("1000000000000")
		res, err := p.GetInputAmountV2(true, amountOut, uint256.Int{})
		require.NoError(t, err)
		require.True(t, res.AmountCalculated.IsZero(), "no input computed for empty-tick pool")
	})
}

// ---------- wordBoundaryTick / floorDiv ----------

// TestFloorDiv covers the rounding Go's / gets wrong for negative ticks. Tick compression rounds
// toward negative infinity on-chain, and getting this backwards puts the boundary in the
// neighbouring bitmap word for every pool priced below tick 0.
func TestFloorDiv(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ a, b, want int }{
		{0, 60, 0},
		{59, 60, 0},
		{60, 60, 1},
		{-1, 60, -1},
		{-59, 60, -1},
		{-60, 60, -1},
		{-61, 60, -2},
		{-887272, 1, -887272},
	} {
		require.Equal(t, tc.want, floorDiv(tc.a, tc.b), "floorDiv(%d, %d)", tc.a, tc.b)
	}
}

// TestWordBoundaryTick mirrors TickBitmap.nextInitializedTickWithinOneWord for the case where the
// word holds no initialized tick: searching down stops at the word's lowest tick, searching up at
// its highest. A word spans 256 compressed ticks, so its span in real ticks is 256*tickSpacing.
func TestWordBoundaryTick(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		tick        int
		tickSpacing int
		zeroForOne  bool
		want        int
	}{
		// Spacing 1: word n covers ticks [256n, 256n+255]. This is the Avalanche dust pool's
		// geometry — its swap starts at tick 291969, inside word 1140 = [291840, 292095].
		{"down from mid-word", 291969, 1, true, 291840},
		{"up from mid-word", 291969, 1, false, 292095},
		{"down from word floor stays put", 291840, 1, true, 291840},
		{"up from word ceiling moves to next word", 292095, 1, false, 292351},

		// Spacing 60: word n covers ticks [15360n, 15360n+15300].
		{"spaced down", 1000, 60, true, 0},
		{"spaced up", 1000, 60, false, 15300},
		{"spaced down from exact multiple", 15360, 60, true, 15360},

		// Negative ticks: the compressed index floors, so tick -1 at spacing 60 lives in word -1
		// alongside tick -15360, not in word 0.
		{"negative down", -1, 60, true, -15360},
		{"negative down from mid-word", -8000, 60, true, -15360},
		// Searching up from -8000 compresses to -134, steps to -133, still in word -1, whose top
		// compressed tick is -1.
		{"negative up", -8000, 60, false, -60},
		// Searching up from -1 steps to compressed 0, which is in word 0, not word -1.
		{"negative up crossing zero", -1, 60, false, 15300},

		// Both ends of the range straddle the bounds; the caller clamps.
		{"below MinTick", -887272, 1, true, -887296},
		{"above MaxTick", 887272, 1, false, 887295},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, wordBoundaryTick(tc.tick, tc.tickSpacing, tc.zeroForOne))
		})
	}
}

// TestWordBoundaryTickSpansOneWord asserts the invariant the swap loop depends on: a step never
// covers more than one bitmap word, which is what lets ComputeSwapStep's own rounding stand in for
// the on-chain rounding without a correction on top.
func TestWordBoundaryTickSpansOneWord(t *testing.T) {
	t.Parallel()

	for _, tickSpacing := range []int{1, 10, 60, 200} {
		for tick := -2000; tick <= 2000; tick++ {
			down := wordBoundaryTick(tick, tickSpacing, true)
			up := wordBoundaryTick(tick, tickSpacing, false)

			require.LessOrEqual(t, down, tick, "downward boundary must not overshoot")
			require.Greater(t, up, tick, "upward boundary must make progress")

			compressed := floorDiv(tick, tickSpacing)
			require.Equal(t, compressed>>8, floorDiv(down, tickSpacing)>>8,
				"downward boundary left the word: tick=%d spacing=%d", tick, tickSpacing)
			require.Equal(t, (compressed+1)>>8, floorDiv(up, tickSpacing)>>8,
				"upward boundary left the word: tick=%d spacing=%d", tick, tickSpacing)
		}
	}
}

// ---------- nextInitializedTickWithinOneWord ----------

// TestNextInitializedTickWithinOneWord covers the wrapper the swap loop actually calls. It is the
// Go counterpart of TickBitmap.nextInitializedTickWithinOneWord: an initialized tick is only
// visible if it shares the bitmap word with the current tick, and running out of ticks in the
// direction of travel yields the word edge rather than an error, because on-chain that condition is
// just an empty bitmap word.
func TestNextInitializedTickWithinOneWord(t *testing.T) {
	t.Parallel()

	liq := uint256.NewInt(500)
	// Spacing 60, so a word spans 15360 ticks. 1200 and 2400 share word 0 ([0, 15300]);
	// 20040 sits in word 1.
	ticks := []TickU256{
		{1200, liq, int256.MustFromDec("500")},
		{2400, liq, int256.MustFromDec("-500")},
		{20040, liq, int256.MustFromDec("500")},
		{21000, liq, int256.MustFromDec("-500")},
	}

	for _, tc := range []struct {
		name       string
		tick       int
		lte        bool
		wantTick   int
		wantInit   bool
		wantHasPos bool
	}{
		// Same word: the initialized tick wins over the edge.
		{"down to initialized tick in word", 2500, true, 2400, true, true},
		{"up to initialized tick in word", 1300, false, 2400, true, true},

		// Same word, further down it: 20040 shares word 1 with 20100, so it stays visible.
		{"down sees far tick in same word", 20100, true, 20040, true, true},

		// Different word: the edge wins, and the stop is not an initialized tick. From 16000
		// (word 1) the nearest tick below is 2400, back in word 0 and therefore invisible.
		{"down stops at word edge", 16000, true, 15360, false, false},
		{"up stops at word edge", 2500, false, 15300, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotTick, gotPos, gotInit, err := nextInitializedTickWithinOneWord(ticks, tc.tick, 60, tc.lte)
			require.NoError(t, err)
			require.Equal(t, tc.wantTick, gotTick)
			require.Equal(t, tc.wantInit, gotInit)
			if tc.wantHasPos {
				require.NotEqual(t, noSlicePos, gotPos)
				require.Equal(t, tc.wantTick, ticks[gotPos].Index, "slicePos must address the returned tick")
			} else {
				require.Equal(t, noSlicePos, gotPos)
			}
		})
	}
}

// TestNextInitializedTickWithinOneWordPastListEnds pins the one place this wrapper deliberately
// departs from the contract. Where the SDK answers an exhausted tick list with the word edge, the
// range errors are propagated: reaching them means the tick list does not cover the current tick,
// which here signals incomplete tracker data far more often than a genuinely empty bitmap, and
// walking on at an assumed liquidity would turn that into a confident wrong quote.
//
// Word-bounded stepping must not quietly swallow them, which is what a fallback would do.
func TestNextInitializedTickWithinOneWordPastListEnds(t *testing.T) {
	t.Parallel()

	liq := uint256.MustFromDecimal("1000000000000000000")
	ticks := []TickU256{
		{1200, liq, int256.MustFromDec("1000000000000000000")},
		{2400, liq, int256.MustFromDec("-1000000000000000000")},
	}

	_, _, _, err := nextInitializedTickWithinOneWord(ticks, 600, 60, true)
	require.ErrorIs(t, err, ErrBelowSmallest)

	_, _, _, err = nextInitializedTickWithinOneWord(ticks, 2400, 60, false)
	require.ErrorIs(t, err, ErrAtOrAboveLargest)

	// And the swap surfaces them rather than returning a quote built on absent ticks.
	var sqrtP uint256.Int
	require.NoError(t, GetSqrtRatioAtTick(600, &sqrtP))
	pool, err := NewPool(FeeMedium, sqrtP, *liq, 600, ticks, 60)
	require.NoError(t, err)

	_, err = pool.GetOutputAmountV2(true, *uint256.NewInt(1e15), uint256.Int{})
	require.ErrorIs(t, err, ErrBelowSmallest)

	// The other direction stays inside the tick list and swaps normally.
	up, err := pool.GetOutputAmountV2(false, *uint256.NewInt(1e15), uint256.Int{})
	require.NoError(t, err)
	require.False(t, up.AmountCalculated.IsZero(), "upward the swap reaches real liquidity")
}
