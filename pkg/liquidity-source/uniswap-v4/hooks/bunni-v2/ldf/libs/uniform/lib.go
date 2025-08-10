package uniform

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// LiquidityDensityX96 computes the liquidity density at a given tick
func LiquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper int) *uint256.Int {
	if roundedTick < tickLower || roundedTick >= tickUpper {
		return uint256.NewInt(0)
	}
	length := (tickUpper - tickLower) / tickSpacing
	if length <= 0 {
		return uint256.NewInt(0)
	}
	return math.DivUp(math.Q96, uint256.NewInt(uint64(length)))
}

// CumulativeAmount0 computes the cumulative amount0
func CumulativeAmount0(tickSpacing, roundedTick int, totalLiquidity *uint256.Int, tickLower, tickUpper int, isCarpet bool) (*uint256.Int, error) {
	if roundedTick >= tickUpper || tickLower >= tickUpper {
		return uint256.NewInt(0), nil
	}
	if roundedTick < tickLower {
		roundedTick = tickLower
	}

	length := (tickUpper - tickLower) / tickSpacing
	if length <= 0 {
		return uint256.NewInt(0), nil
	}

	sqrtPriceRoundedTick, err := math.GetSqrtPriceAtTick(roundedTick)
	if err != nil {
		return nil, err
	}
	sqrtPriceTickUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, err
	}

	// For uniform distribution: totalLiquidity.fullMulX96Up(getAmount0Delta(Q96.divUp(length)))
	// Using the non-carpet version (isCarpet = false)
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))

	amount0, err := math.GetAmount0Delta(
		sqrtPriceRoundedTick,
		sqrtPriceTickUpper,
		liquidityPerTick,
		true, // roundUp
	)
	if err != nil {
		return nil, err
	}

	// Apply the uniform distribution scaling: totalLiquidity.fullMulX96Up(amount0)
	result, err := math.FullMulX96Up(totalLiquidity, amount0)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CumulativeAmount1 computes the cumulative amount1
func CumulativeAmount1(tickSpacing, roundedTick int, totalLiquidity *uint256.Int, tickLower, tickUpper int, isCarpet bool) (*uint256.Int, error) {
	if roundedTick < tickLower || tickLower >= tickUpper {
		return uint256.NewInt(0), nil
	}
	if roundedTick > tickUpper-tickSpacing {
		roundedTick = tickUpper - tickSpacing
	}

	length := (tickUpper - tickLower) / tickSpacing
	if length <= 0 {
		return uint256.NewInt(0), nil
	}

	sqrtPriceTickLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return nil, err
	}
	sqrtPriceRoundedTickPlusSpacing, err := math.GetSqrtPriceAtTick(roundedTick + tickSpacing)
	if err != nil {
		return nil, err
	}

	// For uniform distribution: totalLiquidity.fullMulX96Up(getAmount1Delta(Q96.divUp(length)))
	// Using the non-carpet version (isCarpet = false)
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))

	amount1, err := math.GetAmount1Delta(
		sqrtPriceTickLower,
		sqrtPriceRoundedTickPlusSpacing,
		liquidityPerTick,
		true, // roundUp
	)
	if err != nil {
		return nil, err
	}

	// Apply the uniform distribution scaling: totalLiquidity.fullMulX96Up(amount1)
	result, err := math.FullMulX96Up(totalLiquidity, amount1)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// InverseCumulativeAmount0 computes the inverse of cumulative amount0
func InverseCumulativeAmount0(tickSpacing int, cumulativeAmount0_, totalLiquidity *uint256.Int, tickLower, tickUpper int, isCarpet bool) (bool, int) {
	if cumulativeAmount0_.IsZero() {
		return true, tickUpper
	}

	length := (tickUpper - tickLower) / tickSpacing
	if length <= 0 {
		return false, 0
	}

	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return false, 0
	}

	// For uniform distribution: cumulativeAmount0_.fullMulDiv(Q96, totalLiquidity)
	scaledAmount, err := math.FullMulDiv(cumulativeAmount0_, math.Q96, totalLiquidity)
	if err != nil {
		return false, 0
	}

	// Get next sqrt price from amount0
	// Using Q96.divUp(length) for liquidity
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))
	sqrtPrice, err := math.GetNextSqrtPriceFromAmount0RoundingUp(
		sqrtPriceUpper,
		liquidityPerTick,
		scaledAmount,
		true,
	)
	if err != nil {
		return false, 0
	}

	// Convert sqrt price to tick
	tick, err := math.GetTickAtSqrtPrice(sqrtPrice)
	if err != nil {
		return false, 0
	}

	// Round tick to spacing
	roundedTick := math.RoundTickSingle(tick, tickSpacing)

	// Ensure roundedTick is within valid range
	if roundedTick < tickLower || roundedTick > tickUpper {
		return false, 0
	}

	// Ensure that roundedTick is not tickUpper when cumulativeAmount0_ is non-zero
	if roundedTick == tickUpper {
		return true, tickUpper - tickSpacing
	}

	return true, roundedTick
}

// InverseCumulativeAmount1 computes the inverse of cumulative amount1
func InverseCumulativeAmount1(tickSpacing int, cumulativeAmount1_, totalLiquidity *uint256.Int, tickLower, tickUpper int, isCarpet bool) (bool, int) {
	if cumulativeAmount1_.IsZero() {
		return true, tickLower - tickSpacing
	}

	length := (tickUpper - tickLower) / tickSpacing
	if length <= 0 {
		return false, 0
	}

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return false, 0
	}

	// For uniform distribution: cumulativeAmount1_.fullMulDiv(Q96, totalLiquidity)
	scaledAmount, err := math.FullMulDiv(cumulativeAmount1_, math.Q96, totalLiquidity)
	if err != nil {
		return false, 0
	}

	// Get next sqrt price from amount1
	// Using Q96.divUp(length) for liquidity
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))
	sqrtPrice, err := math.GetNextSqrtPriceFromAmount1RoundingDown(
		sqrtPriceLower,
		liquidityPerTick,
		scaledAmount,
		true,
	)
	if err != nil {
		return false, 0
	}

	// Convert sqrt price to tick
	tick, err := math.GetTickAtSqrtPrice(sqrtPrice)
	if err != nil {
		return false, 0
	}

	// Handle edge case where tick == tickUpper
	if tick == tickUpper {
		tick -= 1
	}

	// Round tick to spacing
	roundedTick := math.RoundTickSingle(tick, tickSpacing)

	// Ensure roundedTick is within valid range
	if roundedTick < tickLower-tickSpacing || roundedTick >= tickUpper {
		return false, 0
	}

	// Ensure that roundedTick is not (tickLower - tickSpacing) when cumulativeAmount1_ is non-zero
	if roundedTick == tickLower-tickSpacing {
		return true, tickLower
	}

	return true, roundedTick
}
