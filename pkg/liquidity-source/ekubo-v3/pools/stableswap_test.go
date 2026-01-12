package pools

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	positionAmount = big256.BONE
	smallAmount    = big256.TenPow(15)
)

func stableswapKey(centerTick int32, amplification uint8) *StableswapPoolKey {
	return NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		NewPoolConfig(
			common.HexToAddress("0x0000000000000000000000000000000000000000"),
			0,
			NewStableswapPoolTypeConfig(centerTick, amplification),
		),
	)
}

func stableswapState(tick int32, liquidity *uint256.Int) *StableswapPoolState {
	return &StableswapPoolState{
		StableswapPoolSwapState: &StableswapPoolSwapState{
			SqrtRatio: math.ToSqrtRatio(tick),
		},
		Liquidity: liquidity,
	}
}

func activeRange(centerTick int32, amplification uint8) (int32, int32) {
	width := math.MaxTick >> amplification
	lower := centerTick - width
	if lower < math.MinTick {
		lower = math.MinTick
	}
	upper := centerTick + width
	if upper > math.MaxTick {
		upper = math.MaxTick
	}
	return lower, upper
}

func setStableswapBounds(pool *StableswapPool, centerTick int32, amplification uint8) {
	lower, upper := activeRange(centerTick, amplification)
	pool.lowerPrice.Set(math.ToSqrtRatio(lower))
	pool.upperPrice.Set(math.ToSqrtRatio(upper))
}

func mintedLiquidity(centerTick int32, amplification uint8, currentTick int32) *uint256.Int {
	lowerTick, upperTick := activeRange(centerTick, amplification)
	sqrtLower := math.ToSqrtRatio(lowerTick)
	sqrtUpper := math.ToSqrtRatio(upperTick)
	sqrtCurrent := math.ToSqrtRatio(currentTick)

	low := uint256.NewInt(0)
	high := big256.UMaxU128.Clone()
	for low.Cmp(high) < 0 {
		diff := new(uint256.Int).Sub(high, low)
		diff.Rsh(diff, 1)
		mid := new(uint256.Int).Add(low, diff)
		mid.AddUint64(mid, 1)

		if withinBudget(mid, sqrtLower, sqrtUpper, sqrtCurrent) {
			low.Set(mid)
		} else {
			high.Set(mid)
			high.SubUint64(high, 1)
		}
	}

	return low
}

func withinBudget(liquidity, sqrtLower, sqrtUpper, sqrtCurrent *uint256.Int) bool {
	needed0, needed1, ok := requiredAmounts(liquidity, sqrtLower, sqrtUpper, sqrtCurrent)
	if !ok {
		return false
	}
	return needed0.Cmp(positionAmount) <= 0 && needed1.Cmp(positionAmount) <= 0
}

func requiredAmounts(
	liquidity, sqrtLower, sqrtUpper, sqrtCurrent *uint256.Int,
) (*uint256.Int, *uint256.Int, bool) {
	if sqrtCurrent.Cmp(sqrtLower) <= 0 {
		needed0, err := math.Amount0Delta(sqrtLower, sqrtUpper, liquidity, true)
		if err != nil {
			return nil, nil, false
		}
		return needed0, big256.U0, true
	}

	if sqrtCurrent.Cmp(sqrtUpper) >= 0 {
		needed1, err := math.Amount1Delta(sqrtLower, sqrtUpper, liquidity, true)
		if err != nil {
			return nil, nil, false
		}
		return big256.U0, needed1, true
	}

	needed0, err := math.Amount0Delta(sqrtCurrent, sqrtUpper, liquidity, true)
	if err != nil {
		return nil, nil, false
	}
	needed1, err := math.Amount1Delta(sqrtLower, sqrtCurrent, liquidity, true)
	if err != nil {
		return nil, nil, false
	}
	return needed0, needed1, true
}

func buildPool(centerTick int32, amplification uint8, currentTick int32) *StableswapPool {
	liquidity := mintedLiquidity(centerTick, amplification, currentTick)
	pool := NewStableswapPool(stableswapKey(centerTick, amplification), stableswapState(currentTick, liquidity))
	setStableswapBounds(pool, centerTick, amplification)
	return pool
}

func quoteAmount(t *testing.T, pool *StableswapPool, isToken1 bool, amount *uint256.Int) *quoting.Quote {
	t.Helper()

	quote, err := pool.Quote(amount, isToken1)
	require.NoError(t, err)

	return quote
}

func TestStableswapPoolQuote(t *testing.T) {
	t.Parallel()

	t.Run("amplification_26_token0_in", func(t *testing.T) {
		pool := buildPool(0, 26, 0)

		quote := quoteAmount(t, pool, false, smallAmount)

		require.Equal(t, smallAmount, quote.ConsumedAmount)
		require.Equal(t, uint256.NewInt(999_999_999_500_000), quote.CalculatedAmount)
	})

	t.Run("amplification_26_token1_in", func(t *testing.T) {
		pool := buildPool(0, 26, 0)

		quote := quoteAmount(t, pool, true, smallAmount)

		require.Equal(t, smallAmount, quote.ConsumedAmount)
		require.Equal(t, uint256.NewInt(999_999_999_500_000), quote.CalculatedAmount)
	})

	t.Run("amplification_1_token0_in", func(t *testing.T) {
		pool := buildPool(0, 1, 0)

		quote := quoteAmount(t, pool, false, smallAmount)

		require.Equal(t, smallAmount, quote.ConsumedAmount)
		require.Equal(t, uint256.NewInt(999_000_999_001_231), quote.CalculatedAmount)
	})

	t.Run("amplification_1_token1_in", func(t *testing.T) {
		pool := buildPool(0, 1, 0)

		quote := quoteAmount(t, pool, true, smallAmount)

		require.Equal(t, smallAmount, quote.ConsumedAmount)
		require.Equal(t, uint256.NewInt(999_000_999_001_231), quote.CalculatedAmount)
	})

	t.Run("outside_range_has_no_liquidity", func(t *testing.T) {
		amplification := uint8(10)
		_, upper := activeRange(0, amplification)
		outsideTick := min(upper+1000, math.MaxTick)

		pool := buildPool(0, amplification, outsideTick)

		quote := quoteAmount(t, pool, true, smallAmount)

		require.True(t, quote.ConsumedAmount.IsZero())
		require.True(t, quote.CalculatedAmount.IsZero())

		swapState, ok := quote.SwapInfo.SwapStateAfter.(*StableswapPoolSwapState)
		require.True(t, ok)
		require.True(t, swapState.SqrtRatio.Cmp(&pool.upperPrice) >= 0)
	})

	t.Run("swap_through_range_boundary", func(t *testing.T) {
		amplification := uint8(10)
		lower, upper := activeRange(0, amplification)
		startTick := upper - 100
		pool := buildPool(0, amplification, startTick)

		quote := quoteAmount(t, pool, false, big256.BONE)

		require.False(t, quote.ConsumedAmount.IsZero())
		require.False(t, quote.CalculatedAmount.IsZero())
		swapState, ok := quote.SwapInfo.SwapStateAfter.(*StableswapPoolSwapState)
		require.True(t, ok)
		require.True(t, swapState.SqrtRatio.Cmp(math.ToSqrtRatio(lower+10)) <= 0)
	})

	t.Run("inside_range_has_liquidity", func(t *testing.T) {
		amplification := uint8(10)
		lower, upper := activeRange(0, amplification)
		midTick := (lower + upper) / 2
		pool := buildPool(0, amplification, midTick)

		quote := quoteAmount(t, pool, false, smallAmount)

		require.Equal(t, smallAmount, quote.ConsumedAmount)
		require.True(t, quote.CalculatedAmount.Cmp(uint256.NewInt(0)) > 0)
		require.True(t, quote.CalculatedAmount.Cmp(smallAmount) < 0)
	})
}
