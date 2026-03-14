package hiddenocean

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func newTestPool(t *testing.T, sqrtPriceX96, liquidity, sqrtPaX96, sqrtPbX96 string, fee uint32) *PoolSimulator {
	t.Helper()

	extra := Extra{
		SqrtPriceX96: uint256.MustFromDecimal(sqrtPriceX96),
		Liquidity:    uint256.MustFromDecimal(liquidity),
		Fee:          fee,
		SqrtPaX96:    uint256.MustFromDecimal(sqrtPaX96),
		SqrtPbX96:    uint256.MustFromDecimal(sqrtPbX96),
	}

	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:   "0xpooladdress",
		Exchange:  "hidden-ocean",
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Swappable: true},
			{Address: "0xtoken1", Swappable: true},
		},
		Reserves: []string{
			"1000000000000000000000", // 1000e18 token0
			"1000000000000000000000", // 1000e18 token1
		},
		Extra: string(extraBytes),
	}

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool})
	require.NoError(t, err)

	return sim
}

func TestCalcAmountOut_ZeroForOne(t *testing.T) {
	// sqrtPriceX96 ~ price=1.0 (79228162514264337593543950336 = sqrt(1) * 2^96)
	// liquidity = 10^18
	// Fee = 3000 (0.3%)
	// Range: sqrtPa slightly below, sqrtPb slightly above
	sqrtPrice := "79228162514264337593543950336"
	liq := "1000000000000000000"
	sqrtPa := "74505858973476688699204988068" // ~price 0.885
	sqrtPb := "84377791587915437410404619782" // ~price 1.133

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: big.NewInt(1000000), // small swap
		},
		TokenOut: "0xtoken1",
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "amountOut should be positive")
	assert.True(t, result.Fee.Amount.Sign() >= 0, "fee should be non-negative")
	assert.Equal(t, int64(150000), result.Gas)

	t.Logf("amountIn=1000000, amountOut=%s, fee=%s",
		result.TokenAmountOut.Amount.String(), result.Fee.Amount.String())
}

func TestCalcAmountOut_OneForZero(t *testing.T) {
	sqrtPrice := "79228162514264337593543950336"
	liq := "1000000000000000000"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken1",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xtoken0",
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "amountOut should be positive")

	t.Logf("amountIn=1000000, amountOut=%s, fee=%s",
		result.TokenAmountOut.Amount.String(), result.Fee.Amount.String())
}

func TestCalcAmountOut_ZeroLiquidity(t *testing.T) {
	sqrtPrice := "79228162514264337593543950336"
	liq := "0"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xtoken1",
	})

	assert.ErrorIs(t, err, ErrZeroLiquidity)
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	sqrtPrice := "79228162514264337593543950336"
	liq := "1000000000000000000"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xinvalid",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xtoken1",
	})

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_PriceAtBoundary(t *testing.T) {
	// Price at lower boundary — no room for zeroForOne swap
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPa, "1000000000000000000", sqrtPa, sqrtPb, 3000)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xtoken1",
	})

	// Price at lower bound, zeroForOne can't go lower
	assert.Error(t, err)

	// But oneForZero should work
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken1",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xtoken0",
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0)
}

func TestUpdateBalance(t *testing.T) {
	sqrtPrice := "79228162514264337593543950336"
	liq := "1000000000000000000"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	originalPrice := new(uint256.Int).Set(sim.sqrtPriceX96)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xtoken1",
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken1", Amount: result.TokenAmountOut.Amount},
		Fee:            pool.TokenAmount{Token: "0xtoken0", Amount: result.Fee.Amount},
	})

	// Price should have decreased (zeroForOne pushes price down)
	assert.True(t, sim.sqrtPriceX96.Cmp(originalPrice) < 0,
		"price should decrease for zeroForOne swap")

	// Reserves should be updated: input reserve increases by (amountIn - fee),
	// since fees are transferred to feeReceiver and not kept in the pool.
	initReserve, _ := new(big.Int).SetString("1000000000000000000000", 10)
	amountInLessFee := new(big.Int).Sub(big.NewInt(1000000), result.Fee.Amount)
	expectedReserve0 := new(big.Int).Add(initReserve, amountInLessFee)
	expectedReserve1 := new(big.Int).Sub(new(big.Int).Set(initReserve), result.TokenAmountOut.Amount)
	assert.Equal(t, expectedReserve0.String(), sim.Info.Reserves[0].String())
	assert.Equal(t, expectedReserve1.String(), sim.Info.Reserves[1].String())
}

func TestCloneState(t *testing.T) {
	sqrtPrice := "79228162514264337593543950336"
	liq := "1000000000000000000"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	cloned := sim.CloneState().(*PoolSimulator)

	// Modify original
	sim.sqrtPriceX96 = uint256.NewInt(12345)
	sim.Info.Reserves[0] = big.NewInt(999)

	// Clone should be unaffected
	assert.Equal(t, sqrtPrice, cloned.sqrtPriceX96.Dec())
	assert.Equal(t, "1000000000000000000000", cloned.Info.Reserves[0].String())
}

func TestCalcAmountOut_LargeSwap(t *testing.T) {
	// Test a swap that exhausts most of the range
	sqrtPrice := "79228162514264337593543950336"
	liq := "1000000000000000000"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	// Very large input — should be partially consumed with remainder
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil), // 10^30
		},
		TokenOut: "0xtoken1",
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0)

	// Should have remaining input (couldn't consume it all within range)
	require.NotNil(t, result.RemainingTokenAmountIn)
	assert.True(t, result.RemainingTokenAmountIn.Amount.Sign() > 0,
		"should have unconsumed input for a very large swap")

	t.Logf("amountOut=%s, fee=%s", result.TokenAmountOut.Amount.String(), result.Fee.Amount.String())
	t.Logf("remaining=%s", result.RemainingTokenAmountIn.Amount.String())
}

// ---------------------------------------------------------------------------
// NewPoolSimulator edge cases
// ---------------------------------------------------------------------------

func TestNewPoolSimulator_StalePool(t *testing.T) {
	extra := Extra{
		SqrtPriceX96: uint256.MustFromDecimal("79228162514264337593543950336"),
		Liquidity:    uint256.MustFromDecimal("1000000000000000000"),
		Fee:          3000,
		SqrtPaX96:    uint256.MustFromDecimal("74505858973476688699204988068"),
		SqrtPbX96:    uint256.MustFromDecimal("84377791587915437410404619782"),
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:   "0xpool",
		Exchange:  "hidden-ocean",
		Type:      DexType,
		Timestamp: time.Now().Add(-60 * time.Second).Unix(), // 60s ago, MaxAge is 30s
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Swappable: true},
			{Address: "0xtoken1", Swappable: true},
		},
		Reserves: []string{"1000", "1000"},
		Extra:    string(extraBytes),
	}

	_, err = NewPoolSimulator(pool.FactoryParams{
		EntityPool: entityPool,
		Opts:       pool.FactoryOpts{StaleCheck: true},
	})
	assert.ErrorIs(t, err, ErrPoolStateStale)
}

func TestNewPoolSimulator_InvalidExtra(t *testing.T) {
	entityPool := entity.Pool{
		Address:   "0xpool",
		Exchange:  "hidden-ocean",
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Swappable: true},
			{Address: "0xtoken1", Swappable: true},
		},
		Reserves: []string{"1000", "1000"},
		Extra:    "invalid json",
	}

	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool})
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// CalcAmountOut edge cases
// ---------------------------------------------------------------------------

func TestCalcAmountOut_NilAmount(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken0", Amount: nil},
		TokenOut:      "0xtoken1",
	})
	assert.ErrorIs(t, err, ErrZeroAmountIn)
}

func TestCalcAmountOut_NilLiquidity(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)
	sim.liquidity = nil

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken1",
	})
	assert.ErrorIs(t, err, ErrZeroLiquidity)
}

func TestCalcAmountOut_PriceBelowRange(t *testing.T) {
	// sqrtPriceX96 below sqrtPaX96 → clamped up to sqrtPaX96
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"
	belowPa := "70000000000000000000000000000"

	sim := newTestPool(t, belowPa, "1000000000000000000", sqrtPa, sqrtPb, 3000)

	// After clamping to sqrtPa, zeroForOne fails (at lower bound)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken1",
	})
	assert.ErrorIs(t, err, ErrNoSwapLimit)

	// oneForZero should work (clamped price == sqrtPa < sqrtPb)
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken0",
	})
	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0)
}

func TestCalcAmountOut_PriceAboveRange(t *testing.T) {
	// sqrtPriceX96 above sqrtPbX96 → clamped down to sqrtPbX96
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"
	abovePb := "90000000000000000000000000000"

	sim := newTestPool(t, abovePb, "1000000000000000000", sqrtPa, sqrtPb, 3000)

	// After clamping to sqrtPb, oneForZero fails (at upper bound)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken0",
	})
	assert.ErrorIs(t, err, ErrNoSwapLimit)

	// zeroForOne should work (clamped price == sqrtPb > sqrtPa)
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken1",
	})
	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0)
}

func TestCalcAmountOut_PriceAtUpperBoundary(t *testing.T) {
	// Price at upper boundary — no room for oneForZero swap
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPb, "1000000000000000000", sqrtPa, sqrtPb, 3000)

	// oneForZero can't go higher
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken0",
	})
	assert.ErrorIs(t, err, ErrNoSwapLimit)

	// zeroForOne should work
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken1",
	})
	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0)
}

// ---------------------------------------------------------------------------
// UpdateBalance edge cases
// ---------------------------------------------------------------------------

func TestUpdateBalance_OneForZero(t *testing.T) {
	sqrtPrice := "79228162514264337593543950336"
	liq := "1000000000000000000"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)
	originalPrice := new(uint256.Int).Set(sim.sqrtPriceX96)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken0",
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(1000000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken0", Amount: result.TokenAmountOut.Amount},
		Fee:            pool.TokenAmount{Token: "0xtoken1", Amount: result.Fee.Amount},
	})

	// Price should have increased (oneForZero pushes price up)
	assert.True(t, sim.sqrtPriceX96.Cmp(originalPrice) > 0,
		"price should increase for oneForZero swap")
}

func TestUpdateBalance_PriceBelowRange(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	// Manually move price below sqrtPa to trigger clamping in UpdateBalance
	sim.sqrtPriceX96 = uint256.MustFromDecimal("70000000000000000000000000000")

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(1000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(900)},
		Fee:            pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(3)},
	})

	// Price was clamped up from below sqrtPa; oneForZero pushes it further up
	assert.True(t, sim.sqrtPriceX96.Cmp(uint256.MustFromDecimal("70000000000000000000000000000")) > 0,
		"price should be above the original below-range value")
}

func TestUpdateBalance_PriceAboveRange(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	// Manually move price above sqrtPb to trigger clamping in UpdateBalance
	sim.sqrtPriceX96 = uint256.MustFromDecimal("90000000000000000000000000000")

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(900)},
		Fee:            pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(3)},
	})

	// Price was clamped down from above sqrtPb; zeroForOne pushes it lower
	assert.True(t, sim.sqrtPriceX96.Cmp(uint256.MustFromDecimal("90000000000000000000000000000")) < 0,
		"price should be below the original above-range value")
}

func TestUpdateBalance_ErrorFallback_ZeroForOne(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	expectedFallback := new(uint256.Int).Set(sim.sqrtPaX96)

	// Set liquidity to zero so GetNextSqrtPriceFromInput errors
	sim.liquidity = uint256.NewInt(0)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(900)},
		Fee:            pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(3)},
	})

	// zeroForOne error fallback sets price to sqrtPaX96
	assert.Equal(t, expectedFallback.Dec(), sim.sqrtPriceX96.Dec())
}

func TestUpdateBalance_ErrorFallback_OneForZero(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	expectedFallback := new(uint256.Int).Set(sim.sqrtPbX96)

	// Set liquidity to zero so GetNextSqrtPriceFromInput errors
	sim.liquidity = uint256.NewInt(0)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(1000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(900)},
		Fee:            pool.TokenAmount{Token: "0xtoken1", Amount: big.NewInt(3)},
	})

	// oneForZero error fallback sets price to sqrtPbX96
	assert.Equal(t, expectedFallback.Dec(), sim.sqrtPriceX96.Dec())
}

func TestUpdateBalance_ConsecutiveSwaps(t *testing.T) {
	sqrtPrice := "79228162514264337593543950336"
	// Use liquidity large enough relative to reserves (1000e18) so recomputeRange
	// doesn't collapse the range. L = 1e24 keeps sqrtPa/sqrtPb viable.
	liq := "1000000000000000000000000"
	sqrtPa := "74505858973476688699204988068"
	sqrtPb := "84377791587915437410404619782"

	sim := newTestPool(t, sqrtPrice, liq, sqrtPa, sqrtPb, 3000)

	// First swap: zeroForOne
	r1, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken1",
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken1", Amount: r1.TokenAmountOut.Amount},
		Fee:            pool.TokenAmount{Token: "0xtoken0", Amount: r1.Fee.Amount},
	})

	priceAfterFirst := new(uint256.Int).Set(sim.sqrtPriceX96)

	// Second swap: same direction, pool should still work
	r2, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtoken1",
	})
	require.NoError(t, err)
	assert.True(t, r2.TokenAmountOut.Amount.Sign() > 0)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtoken0", Amount: big.NewInt(1000000)},
		TokenAmountOut: pool.TokenAmount{Token: "0xtoken1", Amount: r2.TokenAmountOut.Amount},
		Fee:            pool.TokenAmount{Token: "0xtoken0", Amount: r2.Fee.Amount},
	})

	// Price should continue decreasing
	assert.True(t, sim.sqrtPriceX96.Cmp(priceAfterFirst) < 0,
		"price should continue decreasing after second zeroForOne swap")
}

// ---------------------------------------------------------------------------
// recomputeRange edge cases
// ---------------------------------------------------------------------------

func TestRecomputeRange_ZeroLiquidity(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	originalPa := new(uint256.Int).Set(sim.sqrtPaX96)
	originalPb := new(uint256.Int).Set(sim.sqrtPbX96)

	sim.liquidity = uint256.NewInt(0)
	sim.recomputeRange()

	// Should return early, no changes
	assert.Equal(t, originalPa.Dec(), sim.sqrtPaX96.Dec())
	assert.Equal(t, originalPb.Dec(), sim.sqrtPbX96.Dec())
}

func TestRecomputeRange_NilLiquidity(t *testing.T) {
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	originalPa := new(uint256.Int).Set(sim.sqrtPaX96)
	originalPb := new(uint256.Int).Set(sim.sqrtPbX96)

	sim.liquidity = nil
	sim.recomputeRange()

	assert.Equal(t, originalPa.Dec(), sim.sqrtPaX96.Dec())
	assert.Equal(t, originalPb.Dec(), sim.sqrtPbX96.Dec())
}

func TestRecomputeRange_LargeBalance1_PaCollapses(t *testing.T) {
	// balance1 * Q96 / L >= sqrtP → sqrtPaX96 is set to sqrtP
	// Use L = 1000000, balance1 = 2000000.
	// yOverL_Q96 = 2000000 * Q96 / 1000000 = 2 * Q96 ≈ 1.58e29 >> sqrtP ≈ 7.92e28
	sim := newTestPool(t, "79228162514264337593543950336", "1000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	sim.Info.Reserves[0] = big.NewInt(0)
	sim.Info.Reserves[1] = big.NewInt(2000000)

	sim.recomputeRange()

	// sqrtPaX96 should collapse to sqrtPriceX96
	assert.Equal(t, sim.sqrtPriceX96.Dec(), sim.sqrtPaX96.Dec(),
		"sqrtPa should collapse to sqrtP when yOverL_Q96 >= sqrtP")
}

func TestRecomputeRange_SqrtPaBelowMinRatio(t *testing.T) {
	// sqrtP just above minSqrtRatio, small balance1 pushes sqrtPa below it.
	// sqrtP = minSqrtRatio + 100 = 4295128839
	// L = Q96 = 79228162514264337593543950336, balance1 = 200
	// yOverL_Q96 = 200 * Q96 / Q96 = 200
	// sqrtPa = 4295128839 - 200 = 4295128639 < minSqrtRatio (4295128739) → clamped
	sqrtP := "4295128839"
	liq := "79228162514264337593543950336" // Q96

	sim := newTestPool(t, sqrtP, liq, "4295128739", "5000000000", 3000)

	sim.Info.Reserves[0] = big.NewInt(0)
	sim.Info.Reserves[1] = big.NewInt(200)

	sim.recomputeRange()

	assert.Equal(t, minSqrtRatioU256.Dec(), sim.sqrtPaX96.Dec(),
		"sqrtPa should be clamped to minSqrtRatio")
}

func TestRecomputeRange_LargeBalance0_PbCollapses(t *testing.T) {
	// balance0 * sqrtP / Q96 >= L → sqrtPbX96 is set to sqrtP
	// Use sqrtP = Q96 (price=1), L = 1, balance0 = 2.
	// xTimesSqrtP = 2 * Q96 / Q96 = 2 >= L = 1 → sqrtPb = sqrtP
	sim := newTestPool(t, "79228162514264337593543950336", "1",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	sim.Info.Reserves[0] = big.NewInt(2)
	sim.Info.Reserves[1] = big.NewInt(0)

	sim.recomputeRange()

	// With large balance0, sqrtPb collapses to sqrtP.
	// Also yOverL_Q96 = 0 * Q96 / 1 = 0 < sqrtP, so sqrtPa = sqrtP - 0 = sqrtP.
	// pa == pb == sqrtP → pa >= pb guard fires, both set to sqrtP
	assert.Equal(t, sim.sqrtPriceX96.Dec(), sim.sqrtPbX96.Dec())
	assert.Equal(t, sim.sqrtPriceX96.Dec(), sim.sqrtPaX96.Dec())
}

func TestRecomputeRange_SqrtPbAboveMax(t *testing.T) {
	// (L * sqrtP) / (L - xTimesSqrtP) > maxSqrtRatio - 1 → clamped
	// Use sqrtP = Q96, L = Q96, balance0 = Q96 - 1.
	// xTimesSqrtP = (Q96-1) * Q96 / Q96 = Q96 - 1
	// denom = Q96 - (Q96 - 1) = 1
	// sqrtPb = Q96 * Q96 / 1 = Q96^2 ≈ 6.28e57 >> maxSqrtRatio → clamped
	q96Str := "79228162514264337593543950336"
	q96Minus1 := new(big.Int).Sub(uint256.MustFromDecimal(q96Str).ToBig(), big.NewInt(1))

	sim := newTestPool(t, q96Str, q96Str, "4295128739", q96Str, 3000)

	sim.Info.Reserves[0] = q96Minus1
	sim.Info.Reserves[1] = big.NewInt(0)

	sim.recomputeRange()

	maxMinusOne := new(uint256.Int).Sub(maxSqrtRatioU256, uint256.NewInt(1))
	assert.Equal(t, maxMinusOne.Dec(), sim.sqrtPbX96.Dec(),
		"sqrtPb should be clamped to maxSqrtRatio - 1")
}

func TestRecomputeRange_SqrtPbNearSqrtP(t *testing.T) {
	// balance0 = 0 → xTimesSqrtP = 0, sqrtPb = L*sqrtP/L = sqrtP, which is <= sqrtP+1 → clamped to sqrtP+1
	sim := newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	sim.Info.Reserves[0] = big.NewInt(0)
	sim.Info.Reserves[1] = big.NewInt(0)

	sim.recomputeRange()

	sqrtPPlus1 := new(uint256.Int).Add(sim.sqrtPriceX96, uint256.NewInt(1))
	assert.Equal(t, sqrtPPlus1.Dec(), sim.sqrtPbX96.Dec(),
		"sqrtPb should be clamped to sqrtP + 1 when balance0 = 0")
}

func TestRecomputeRange_PaGePb_BothCollapse(t *testing.T) {
	// Both pa and pb collapse to sqrtP, triggering the pa >= pb guard.
	// Use L = 1, balance1 = 2 (forces pa = sqrtP), balance0 = 2 (forces pb = sqrtP).
	sim := newTestPool(t, "79228162514264337593543950336", "1",
		"74505858973476688699204988068", "84377791587915437410404619782", 3000)

	sim.Info.Reserves[0] = big.NewInt(2)
	sim.Info.Reserves[1] = big.NewInt(2)

	sim.recomputeRange()

	assert.Equal(t, sim.sqrtPriceX96.Dec(), sim.sqrtPaX96.Dec(),
		"pa should collapse to sqrtP")
	assert.Equal(t, sim.sqrtPriceX96.Dec(), sim.sqrtPbX96.Dec(),
		"pb should collapse to sqrtP")
}

// ---------------------------------------------------------------------------
// uint256FromBigInt
// ---------------------------------------------------------------------------

func TestUint256FromBigInt(t *testing.T) {
	t.Run("nil returns zero", func(t *testing.T) {
		result := uint256FromBigInt(nil)
		assert.True(t, result.IsZero())
	})

	t.Run("zero", func(t *testing.T) {
		result := uint256FromBigInt(big.NewInt(0))
		assert.True(t, result.IsZero())
	})

	t.Run("positive value", func(t *testing.T) {
		result := uint256FromBigInt(big.NewInt(12345))
		assert.Equal(t, "12345", result.Dec())
	})

	t.Run("large value", func(t *testing.T) {
		v, _ := new(big.Int).SetString("79228162514264337593543950336", 10)
		result := uint256FromBigInt(v)
		assert.Equal(t, "79228162514264337593543950336", result.Dec())
	})
}
