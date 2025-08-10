package doublegeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/geometric"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// CumulativeAmount0 computes the cumulative amount0 using two geometric distributions
func CumulativeAmount0(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	minTick,
	length0,
	length1 int,
	alpha0X96,
	alpha1X96,
	weight0,
	weight1 *uint256.Int,
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
	amount0_0, err := geoLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		&totalLiquidity0,
		minTick+length1*tickSpacing,
		length0,
		alpha0X96,
	)
	if err != nil {
		return nil, err
	}

	// Calculate cumulative amount0 for distribution 1 (left distribution)
	amount0_1, err := geoLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		&totalLiquidity1,
		minTick,
		length1,
		alpha1X96,
	)
	if err != nil {
		return nil, err
	}

	// Sum the amounts
	var result uint256.Int
	result.Add(amount0_0, amount0_1)
	return &result, nil
}

// CumulativeAmount1 computes the cumulative amount1 using two geometric distributions
func CumulativeAmount1(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	minTick,
	length0,
	length1 int,
	alpha0X96,
	alpha1X96,
	weight0,
	weight1 *uint256.Int,
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
	amount1_0, err := geoLib.CumulativeAmount1(tickSpacing, roundedTick, &totalLiquidity0, minTick+length1*tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	// Calculate cumulative amount1 for distribution 1 (left distribution)
	amount1_1, err := geoLib.CumulativeAmount1(tickSpacing, roundedTick, &totalLiquidity1, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	// Sum the amounts
	var result uint256.Int
	result.Add(amount1_0, amount1_1)
	return &result, nil
}

// InverseCumulativeAmount0 computes the inverse cumulative amount0
func InverseCumulativeAmount0(
	tickSpacing int,
	cumulativeAmount0_,
	totalLiquidity *uint256.Int,
	minTick,
	length0,
	length1 int,
	alpha0X96,
	alpha1X96,
	weight0,
	weight1 *uint256.Int,
) (bool, int, error) {
	minTick0 := minTick + length1*tickSpacing
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity0 uint256.Int
	totalLiquidity0.Mul(totalLiquidity, weight0)
	totalLiquidity0.Div(&totalLiquidity0, &totalWeight)

	ldf0CumulativeAmount0, err := geoLib.CumulativeAmount0(tickSpacing, minTick0, &totalLiquidity0, minTick0, length0, alpha0X96)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount0_.Cmp(ldf0CumulativeAmount0) <= 0 {
		return geoLib.InverseCumulativeAmount0(tickSpacing, cumulativeAmount0_, &totalLiquidity0, minTick0, length0, alpha0X96)
	} else {
		var remainder uint256.Int
		remainder.Sub(cumulativeAmount0_, ldf0CumulativeAmount0)

		var totalLiquidity1 uint256.Int
		totalLiquidity1.Mul(totalLiquidity, weight1)
		totalLiquidity1.Div(&totalLiquidity1, &totalWeight)

		return geoLib.InverseCumulativeAmount0(tickSpacing, &remainder, &totalLiquidity1, minTick, length1, alpha1X96)
	}
}

// inverseCumulativeAmount1 computes the inverse cumulative amount1
func InverseCumulativeAmount1(tickSpacing int, cumulativeAmount1_, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (bool, int, error) {
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity1 uint256.Int
	totalLiquidity1.Mul(totalLiquidity, weight1)
	totalLiquidity1.Div(&totalLiquidity1, &totalWeight)

	ldf1CumulativeAmount1, err := geoLib.CumulativeAmount1(tickSpacing, minTick+length1*tickSpacing, &totalLiquidity1, minTick, length1, alpha1X96)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount1_.Cmp(ldf1CumulativeAmount1) <= 0 {
		return geoLib.InverseCumulativeAmount1(tickSpacing, cumulativeAmount1_, &totalLiquidity1, minTick, length1, alpha1X96)
	} else {
		var remainder uint256.Int
		remainder.Sub(cumulativeAmount1_, ldf1CumulativeAmount1)

		var totalLiquidity0 uint256.Int
		totalLiquidity0.Mul(totalLiquidity, weight0)
		totalLiquidity0.Div(&totalLiquidity0, &totalWeight)

		return geoLib.InverseCumulativeAmount1(tickSpacing, &remainder, &totalLiquidity0, minTick+length1*tickSpacing, length0, alpha0X96)
	}
}

// LiquidityDensityX96 computes the liquidity density using weighted sum of two geometric distributions
func LiquidityDensityX96(tickSpacing, roundedTick, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (*uint256.Int, error) {
	// Calculate density for distribution 0 (right distribution)
	density0, err := geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick+length1*tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	// Calculate density for distribution 1 (left distribution)
	density1, err := geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	// Apply weighted sum
	result := math.WeightedSum(density0, weight0, density1, weight1)
	return result, nil
}
