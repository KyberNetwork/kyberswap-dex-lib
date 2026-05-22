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
	sqrtPrice, _ := uint256.FromDecimal("4082682361430349352208957440")
	liquidity, _ := uint256.FromDecimal("461286494113032089234462")

	pool, err := newPool(FeeAmount(3000), sqrtPrice, liquidity, -59315, ticks, 60)
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
		require.ErrorIs(t, err, errZeroTickSpacing)
	})

	t.Run("bad spacing alignment", func(t *testing.T) {
		ticks := []TickU256{
			{-61, liq, net}, // not a multiple of 60
			{61, liq, netNeg},
		}
		err := validateList(ticks, 60)
		require.ErrorIs(t, err, errInvalidTickSpacing)
	})

	t.Run("non-zero net sum", func(t *testing.T) {
		ticks := []TickU256{
			{-60, liq, net},
			{60, liq, net}, // net should be negative to sum to zero
		}
		err := validateList(ticks, 60)
		require.ErrorIs(t, err, errZeroNet)
	})

	t.Run("unsorted", func(t *testing.T) {
		ticks := []TickU256{
			{60, liq, net},
			{-60, liq, netNeg},
		}
		err := validateList(ticks, 60)
		require.ErrorIs(t, err, errSorted)
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
	require.ErrorIs(t, err, errBelowSmallest)

	// at or above largest (lte=false) → error
	_, _, err = nextInitializedTickIndex(ticks, 120, false)
	require.ErrorIs(t, err, errAtOrAboveLargest)
}

// ---------- newPool ----------

func TestNewPool(t *testing.T) {
	t.Parallel()

	ticks := []TickU256{
		{-60, uint256.NewInt(100), int256.NewInt(100)},
		{60, uint256.NewInt(100), int256.MustFromDec("-100")},
	}

	// valid pool at tick 0: sqrtPrice must be between sqrtRatioAtTick(0) and sqrtRatioAtTick(1)
	// sqrtRatioAtTick(0) = 79228162514264337593543950336 (Q96)
	sqrtPrice, _ := uint256.FromDecimal("79228162514264337593543950336")
	liquidity := uint256.NewInt(1e9)

	p, err := newPool(FeeMedium, sqrtPrice, liquidity, 0, ticks, 60)
	require.NoError(t, err)
	require.Equal(t, 0, p.TickCurrent)
	require.Equal(t, FeeMedium, p.Fee)

	// fee too high
	_, err = newPool(FeeMax, sqrtPrice, liquidity, 0, ticks, 60)
	require.ErrorIs(t, err, errFeeTooHigh)

	// sqrtPrice out of range for tick
	bad, _ := uint256.FromDecimal("1")
	_, err = newPool(FeeMedium, bad, liquidity, 0, ticks, 60)
	require.ErrorIs(t, err, errInvalidSqrtRatioX96)
}

// ---------- getOutputAmountV2 (exact input) ----------

func TestGetOutputAmountV2(t *testing.T) {
	t.Parallel()
	p := makeSmallPool(t)

	// Swap 1 WETH (1e18) in as token1 (zeroForOne=false, token1→token0)
	amountIn := new(int256.Int)
	amountIn.SetFromBig(new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))

	// price limit: slightly above current sqrt price (going up, token1→token0)
	priceLimit, _ := uint256.FromDecimal("1461446703485210103287273052203988822378723970341") // MaxSqrtRatio - 1

	result, err := p.getOutputAmountV2(amountIn, false, priceLimit)
	require.NoError(t, err)
	require.True(t, result.ReturnedAmount.Sign() > 0, "output must be positive")
	require.NotNil(t, result.SqrtRatioX96)
	require.NotNil(t, result.Liquidity)
}

// ---------- getInputAmountV2 (exact output) ----------

func TestGetInputAmountV2(t *testing.T) {
	t.Parallel()
	p := makeSmallPool(t)

	// We want exactly 1e15 of token0 out (zeroForOne=true, token0 output)
	amountOut := new(int256.Int)
	amountOut.SetFromBig(new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil))

	priceLimit, _ := uint256.FromDecimal("4295128740") // MinSqrtRatio + 1

	amountIn, newState, err := p.getInputAmountV2(amountOut, true, priceLimit)
	require.NoError(t, err)
	require.True(t, amountIn.Sign() > 0, "input amount must be positive")
	require.NotNil(t, newState)
	require.NotNil(t, newState.SqrtRatioX96)

	// Cross-check: feeding the computed amountIn into getOutputAmountV2 should yield
	// at least amountOut (may be slightly more due to rounding convention).
	priceLimit2, _ := uint256.FromDecimal("4295128740")
	outResult, err := p.getOutputAmountV2(amountIn, true, priceLimit2)
	require.NoError(t, err)
	gotOut := outResult.ReturnedAmount.ToBig()
	require.True(t, gotOut.Cmp(amountOut.ToBig()) >= 0,
		"round-trip: out(%s) < requested(%s)", gotOut, amountOut.ToBig())
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
