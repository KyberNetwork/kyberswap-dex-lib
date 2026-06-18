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
	require.NotNil(t, res.SqrtRatioX96)
	require.NotNil(t, res.Liquidity)
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
	require.NotNil(t, sr.SqrtRatioX96)

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
