package doublegeometric

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// cumulativeAmount0 computes the cumulative amount0 using two geometric distributions
func CumulativeAmount0(
	tickSpacing, roundedTick int, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int,
) (*uint256.Int, error) {
	// Calculate total liquidity for each distribution
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity0 uint256.Int
	totalLiquidity0.Mul(totalLiquidity, weight0)
	totalLiquidity0.Div(&totalLiquidity0, &totalWeight)

	var totalLiquidity1 uint256.Int
	totalLiquidity1.Mul(totalLiquidity, weight1)
	totalLiquidity1.Div(&totalLiquidity1, &totalWeight)

	// Calculate cumulative amount0 for distribution 0 (right distribution)
	amount0_0, err := geometricCumulativeAmount0(tickSpacing, roundedTick, &totalLiquidity0, minTick+length1*tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	// Calculate cumulative amount0 for distribution 1 (left distribution)
	amount0_1, err := geometricCumulativeAmount0(tickSpacing, roundedTick, &totalLiquidity1, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	// Sum the amounts
	var result uint256.Int
	result.Add(amount0_0, amount0_1)
	return &result, nil
}

// cumulativeAmount1 computes the cumulative amount1 using two geometric distributions
func CumulativeAmount1(
	tickSpacing, roundedTick int, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int,
) (*uint256.Int, error) {
	// Calculate total liquidity for each distribution
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity0 uint256.Int
	totalLiquidity0.Mul(totalLiquidity, weight0)
	totalLiquidity0.Div(&totalLiquidity0, &totalWeight)

	var totalLiquidity1 uint256.Int
	totalLiquidity1.Mul(totalLiquidity, weight1)
	totalLiquidity1.Div(&totalLiquidity1, &totalWeight)

	// Calculate cumulative amount1 for distribution 0 (right distribution)
	amount1_0, err := geometricCumulativeAmount1(tickSpacing, roundedTick, &totalLiquidity0, minTick+length1*tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	// Calculate cumulative amount1 for distribution 1 (left distribution)
	amount1_1, err := geometricCumulativeAmount1(tickSpacing, roundedTick, &totalLiquidity1, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	// Sum the amounts
	var result uint256.Int
	result.Add(amount1_0, amount1_1)
	return &result, nil
}

// InverseCumulativeAmount0 computes the inverse cumulative amount0
func InverseCumulativeAmount0(tickSpacing int, cumulativeAmount0_, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (bool, int, error) {
	// try ldf0 first, if fails then try ldf1 with remainder
	minTick0 := minTick + length1*tickSpacing
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity0 uint256.Int
	totalLiquidity0.Mul(totalLiquidity, weight0)
	totalLiquidity0.Div(&totalLiquidity0, &totalWeight)

	ldf0CumulativeAmount0, err := geometricCumulativeAmount0(tickSpacing, minTick0, &totalLiquidity0, minTick0, length0, alpha0X96)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount0_.Cmp(ldf0CumulativeAmount0) <= 0 {
		return geometricInverseCumulativeAmount0(tickSpacing, cumulativeAmount0_, &totalLiquidity0, minTick0, length0, alpha0X96)
	} else {
		var remainder uint256.Int
		remainder.Sub(cumulativeAmount0_, ldf0CumulativeAmount0)

		var totalLiquidity1 uint256.Int
		totalLiquidity1.Mul(totalLiquidity, weight1)
		totalLiquidity1.Div(&totalLiquidity1, &totalWeight)

		return geometricInverseCumulativeAmount0(tickSpacing, &remainder, &totalLiquidity1, minTick, length1, alpha1X96)
	}
}

// inverseCumulativeAmount1 computes the inverse cumulative amount1
func InverseCumulativeAmount1(tickSpacing int, cumulativeAmount1_, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (bool, int, error) {
	// try ldf1 first, if fails then try ldf0 with remainder
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity1 uint256.Int
	totalLiquidity1.Mul(totalLiquidity, weight1)
	totalLiquidity1.Div(&totalLiquidity1, &totalWeight)

	ldf1CumulativeAmount1, err := geometricCumulativeAmount1(tickSpacing, minTick+length1*tickSpacing, &totalLiquidity1, minTick, length1, alpha1X96)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount1_.Cmp(ldf1CumulativeAmount1) <= 0 {
		return geometricInverseCumulativeAmount1(tickSpacing, cumulativeAmount1_, &totalLiquidity1, minTick, length1, alpha1X96)
	} else {
		var remainder uint256.Int
		remainder.Sub(cumulativeAmount1_, ldf1CumulativeAmount1)

		var totalLiquidity0 uint256.Int
		totalLiquidity0.Mul(totalLiquidity, weight0)
		totalLiquidity0.Div(&totalLiquidity0, &totalWeight)

		return geometricInverseCumulativeAmount1(tickSpacing, &remainder, &totalLiquidity0, minTick+length1*tickSpacing, length0, alpha0X96)
	}
}

// geometricInverseCumulativeAmount0 computes inverse cumulative amount0 for a single geometric distribution
func geometricInverseCumulativeAmount0(tickSpacing int, cumulativeAmount0_, totalLiquidity *uint256.Int, minTick, length int, alphaX96 *uint256.Int) (bool, int, error) {
	// Simplified binary search implementation
	left := minTick
	right := minTick + length*tickSpacing

	for left < right {
		mid := (left + right) / 2
		mid = (mid / tickSpacing) * tickSpacing // round to tick spacing

		amount0, err := geometricCumulativeAmount0(tickSpacing, mid, totalLiquidity, minTick, length, alphaX96)
		if err != nil {
			return false, 0, err
		}

		if amount0.Cmp(cumulativeAmount0_) >= 0 {
			right = mid
		} else {
			left = mid + tickSpacing
		}
	}

	return true, left, nil
}

// geometricInverseCumulativeAmount1 computes inverse cumulative amount1 for a single geometric distribution
func geometricInverseCumulativeAmount1(tickSpacing int, cumulativeAmount1_, totalLiquidity *uint256.Int, minTick, length int, alphaX96 *uint256.Int) (bool, int, error) {
	// Simplified binary search implementation
	left := minTick
	right := minTick + length*tickSpacing

	for left < right {
		mid := (left + right) / 2
		mid = (mid / tickSpacing) * tickSpacing // round to tick spacing

		amount1, err := geometricCumulativeAmount1(tickSpacing, mid, totalLiquidity, minTick, length, alphaX96)
		if err != nil {
			return false, 0, err
		}

		if amount1.Cmp(cumulativeAmount1_) >= 0 {
			right = mid
		} else {
			left = mid + tickSpacing
		}
	}

	return true, left, nil
}

// geometricCumulativeAmount0 computes cumulative amount0 for a single geometric distribution
func geometricCumulativeAmount0(tickSpacing, roundedTick int, totalLiquidity *uint256.Int, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick >= minTick+length*tickSpacing {
		return uint256.NewInt(0), nil
	}

	var result uint256.Int
	for i := (roundedTick - minTick) / tickSpacing; i < length; i++ {
		density, err := geometricLiquidityDensityX96(tickSpacing, minTick+i*tickSpacing, minTick, length, alphaX96)
		if err != nil {
			return nil, err
		}

		sqrtPriceLower, err := math.GetSqrtPriceAtTick(minTick + i*tickSpacing)
		if err != nil {
			return nil, err
		}
		sqrtPriceUpper, err := math.GetSqrtPriceAtTick(minTick + (i+1)*tickSpacing)
		if err != nil {
			return nil, err
		}

		amount0, err := math.GetAmount0Delta(
			sqrtPriceLower,
			sqrtPriceUpper,
			density,
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		result.Add(&result, amount0)
	}

	return &result, nil
}

// geometricCumulativeAmount1 computes cumulative amount1 for a single geometric distribution
func geometricCumulativeAmount1(tickSpacing, roundedTick int, totalLiquidity *uint256.Int, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick <= minTick {
		return uint256.NewInt(0), nil
	}

	var result uint256.Int
	for i := 0; i < (roundedTick-minTick)/tickSpacing && i < length; i++ {
		density, err := geometricLiquidityDensityX96(tickSpacing, minTick+i*tickSpacing, minTick, length, alphaX96)
		if err != nil {
			return nil, err
		}

		sqrtPriceLower, err := math.GetSqrtPriceAtTick(minTick + i*tickSpacing)
		if err != nil {
			return nil, err
		}
		sqrtPriceUpper, err := math.GetSqrtPriceAtTick(minTick + (i+1)*tickSpacing)
		if err != nil {
			return nil, err
		}

		amount1, err := math.GetAmount1Delta(
			sqrtPriceLower,
			sqrtPriceUpper,
			density,
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		result.Add(&result, amount1)
	}

	return &result, nil
}

// LiquidityDensityX96 computes the liquidity density using weighted sum of two geometric distributions
func LiquidityDensityX96(tickSpacing, roundedTick, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (*uint256.Int, error) {
	// Calculate density for distribution 0 (right distribution)
	density0, err := geometricLiquidityDensityX96(tickSpacing, roundedTick, minTick+length1*tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	// Calculate density for distribution 1 (left distribution)
	density1, err := geometricLiquidityDensityX96(tickSpacing, roundedTick, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	// Apply weighted sum: density0 * weight0 + density1 * weight1
	var weightedDensity0 uint256.Int
	weightedDensity0.Mul(density0, weight0)
	var weightedDensity1 uint256.Int
	weightedDensity1.Mul(density1, weight1)

	var result uint256.Int
	result.Add(&weightedDensity0, &weightedDensity1)
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)
	result.Div(&result, &totalWeight)

	return &result, nil
}

// geometricLiquidityDensityX96 computes the liquidity density for a single geometric distribution
func geometricLiquidityDensityX96(tickSpacing, roundedTick, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick < minTick || roundedTick >= minTick+length*tickSpacing {
		return uint256.NewInt(0), nil
	}

	x := (roundedTick - minTick) / tickSpacing

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		term1, err := math.Rpow(&alphaInvX96, length-x, math.Q96)
		if err != nil {
			return nil, err
		}
		var term2 uint256.Int
		term2.Sub(alphaX96, math.Q96)
		term3, err := math.Rpow(&alphaInvX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(term1, &term2, &denom)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		// alpha <= 1
		var term1 uint256.Int
		term1.Sub(math.Q96, alphaX96)
		term2, err := math.Rpow(alphaX96, x, math.Q96)
		if err != nil {
			return nil, err
		}
		term3, err := math.Rpow(alphaX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(&term1, term2, &denom)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}
