package carpetedgeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/geometric"
	uniformLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/uniform"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

// CumulativeAmount0 computes the cumulative amount0
func CumulativeAmount0(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (*uint256.Int, error) {
	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick, err := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		minTick,
		length,
		weightCarpet,
	)
	if err != nil {
		return nil, err
	}

	// Left carpet amount0
	leftCarpetAmount0, err := uniformLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		leftCarpetLiquidity,
		minUsableTick,
		minTick,
		true,
	)
	if err != nil {
		return nil, err
	}

	// Main geometric amount0
	mainAmount0, err := geoLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		mainLiquidity,
		minTick,
		length,
		alphaX96,
	)
	if err != nil {
		return nil, err
	}

	// Right carpet amount0
	rightCarpetAmount0, err := uniformLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		rightCarpetLiquidity,
		minTick+length*tickSpacing,
		maxUsableTick,
		true,
	)
	if err != nil {
		return nil, err
	}

	// Sum all amounts
	var result uint256.Int
	result.Add(leftCarpetAmount0, mainAmount0)
	result.Add(&result, rightCarpetAmount0)

	return &result, nil
}

// CumulativeAmount1 computes the cumulative amount1
func CumulativeAmount1(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (*uint256.Int, error) {
	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick, err := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		minTick,
		length,
		weightCarpet,
	)
	if err != nil {
		return nil, err
	}

	// Left carpet amount1
	leftCarpetAmount1, err := uniformLib.CumulativeAmount1(
		tickSpacing,
		roundedTick,
		leftCarpetLiquidity,
		minUsableTick,
		minTick,
		true,
	)
	if err != nil {
		return nil, err
	}

	// Main geometric amount1
	mainAmount1, err := geoLib.CumulativeAmount1(
		tickSpacing,
		roundedTick,
		mainLiquidity,
		minTick,
		length,
		alphaX96,
	)
	if err != nil {
		return nil, err
	}

	// Right carpet amount1
	rightCarpetAmount1, err := uniformLib.CumulativeAmount1(
		tickSpacing,
		roundedTick,
		rightCarpetLiquidity,
		minTick+length*tickSpacing,
		maxUsableTick,
		true,
	)
	if err != nil {
		return nil, err
	}

	// Sum all amounts
	var result uint256.Int
	result.Add(leftCarpetAmount1, mainAmount1)
	result.Add(&result, rightCarpetAmount1)

	return &result, nil
}

// getCarpetedLiquidity computes the liquidity distribution for carpeted parts
func getCarpetedLiquidity(
	tickSpacing int,
	totalLiquidity *uint256.Int,
	minTick,
	length int,
	weightCarpet *uint256.Int,
) (
	leftCarpetLiquidity,
	mainLiquidity,
	rightCarpetLiquidity *uint256.Int,
	minUsableTick,
	maxUsableTick int,
	err error,
) {
	minUsableTick = math.MinUsableTick(tickSpacing)
	maxUsableTick = math.MaxUsableTick(tickSpacing)
	numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/tickSpacing - length

	if numRoundedTicksCarpeted <= 0 {
		return u256.U0, totalLiquidity, u256.U0, minUsableTick, maxUsableTick, nil
	}

	// Main liquidity: totalLiquidity * (WAD - weightCarpet) / WAD
	var wadMinusWeightCarpet uint256.Int
	wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
	mainLiquidity, err = math.FullMulDiv(totalLiquidity, &wadMinusWeightCarpet, math.WAD)
	if err != nil {
		return nil, nil, nil, 0, 0, err
	}

	// Carpet liquidity: totalLiquidity - mainLiquidity
	var carpetLiquidity uint256.Int
	carpetLiquidity.Sub(totalLiquidity, mainLiquidity)

	// Right carpet liquidity: carpetLiquidity * rightCarpetNumRoundedTicks / numRoundedTicksCarpeted
	rightCarpetNumRoundedTicks := (maxUsableTick-minTick)/tickSpacing - length
	rightCarpetLiquidity, err = math.FullMulDiv(
		&carpetLiquidity,
		uint256.NewInt(uint64(rightCarpetNumRoundedTicks)),
		uint256.NewInt(uint64(numRoundedTicksCarpeted)),
	)
	if err != nil {
		return nil, nil, nil, 0, 0, err
	}

	// Left carpet liquidity: carpetLiquidity - rightCarpetLiquidity
	var leftCarpetLiquidityVar uint256.Int
	leftCarpetLiquidityVar.Sub(&carpetLiquidity, rightCarpetLiquidity)
	leftCarpetLiquidity = &leftCarpetLiquidityVar

	return
}

// LiquidityDensityX96 computes the liquidity density at a given tick
func LiquidityDensityX96(
	tickSpacing,
	roundedTick,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (*uint256.Int, error) {
	if roundedTick >= minTick && roundedTick < minTick+length*tickSpacing {
		// Inside the main geometric distribution
		geometricDensity, err := geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick, length, alphaX96)
		if err != nil {
			return nil, err
		}

		// Apply carpet weight: geometricDensity * (WAD - weightCarpet) / WAD
		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
		result, err := math.FullMulDiv(geometricDensity, &wadMinusWeightCarpet, math.WAD)
		if err != nil {
			return nil, err
		}

		return result, nil
	} else {
		// Outside the main distribution - use carpet distribution
		minUsableTick := math.MinUsableTick(tickSpacing)
		maxUsableTick := math.MaxUsableTick(tickSpacing)
		numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/tickSpacing - length
		if numRoundedTicksCarpeted <= 0 {
			return u256.U0, nil
		}

		// Main liquidity: Q96 * (WAD - weightCarpet) / WAD
		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
		mainLiquidity, err := math.FullMulDiv(math.Q96, &wadMinusWeightCarpet, math.WAD)
		if err != nil {
			return nil, err
		}

		// Carpet liquidity: Q96 - mainLiquidity
		var carpetLiquidity uint256.Int
		carpetLiquidity.Sub(math.Q96, mainLiquidity)

		// Return carpet liquidity divided by number of carpeted ticks (with rounding up)
		result := math.DivUp(&carpetLiquidity, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
		return result, nil
	}
}

// InverseCumulativeAmount0 computes the inverse of cumulative amount0
// Based on Solidity LibCarpetedGeometricDistribution.inverseCumulativeAmount0
func InverseCumulativeAmount0(
	tickSpacing int,
	cumulativeAmount0_ *uint256.Int,
	totalLiquidity *uint256.Int,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (bool, int, error) {
	if cumulativeAmount0_.IsZero() {
		maxUsableTick := math.MaxUsableTick(tickSpacing)
		return true, maxUsableTick, nil
	}

	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick, err := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		minTick,
		length,
		weightCarpet,
	)
	if err != nil {
		return false, 0, err
	}

	// Try LDFs in the order of right carpet, main, left carpet
	rightCarpetCumulativeAmount0, err := uniformLib.CumulativeAmount0(
		tickSpacing,
		minTick+length*tickSpacing,
		rightCarpetLiquidity,
		minTick+length*tickSpacing,
		maxUsableTick,
		true,
	)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount0_.Cmp(rightCarpetCumulativeAmount0) <= 0 && !rightCarpetLiquidity.IsZero() {
		// Use right carpet
		success, roundedTick := uniformLib.InverseCumulativeAmount0(
			tickSpacing,
			cumulativeAmount0_,
			rightCarpetLiquidity,
			minTick+length*tickSpacing,
			maxUsableTick,
			true,
		)
		return success, roundedTick, nil
	} else {
		var remainder uint256.Int
		remainder.Sub(cumulativeAmount0_, rightCarpetCumulativeAmount0)
		mainCumulativeAmount0, err := geoLib.CumulativeAmount0(
			tickSpacing,
			minTick,
			mainLiquidity,
			minTick,
			length,
			alphaX96,
		)
		if err != nil {
			return false, 0, err
		}

		if remainder.Cmp(mainCumulativeAmount0) <= 0 {
			// Use main
			success, roundedTick, err := geoLib.InverseCumulativeAmount0(
				tickSpacing,
				&remainder,
				mainLiquidity,
				minTick,
				length,
				alphaX96,
			)
			return success, roundedTick, err
		} else if !leftCarpetLiquidity.IsZero() {
			// Use left carpet
			remainder.Sub(&remainder, mainCumulativeAmount0)
			success, roundedTick := uniformLib.InverseCumulativeAmount0(
				tickSpacing,
				&remainder,
				leftCarpetLiquidity,
				minUsableTick,
				minTick,
				true,
			)
			return success, roundedTick, nil
		}
	}
	return false, 0, nil
}

// InverseCumulativeAmount1 computes the inverse of cumulative amount1
// Based on Solidity LibCarpetedGeometricDistribution.inverseCumulativeAmount1
func InverseCumulativeAmount1(
	tickSpacing int,
	cumulativeAmount1_ *uint256.Int,
	totalLiquidity *uint256.Int,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (bool, int, error) {
	if cumulativeAmount1_.IsZero() {
		minUsableTick := math.MinUsableTick(tickSpacing)
		return true, minUsableTick - tickSpacing, nil
	}

	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick, err := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		minTick,
		length,
		weightCarpet,
	)
	if err != nil {
		return false, 0, err
	}

	// Try LDFs in the order of left carpet, main, right carpet
	leftCarpetCumulativeAmount1, err := uniformLib.CumulativeAmount1(
		tickSpacing,
		minTick,
		leftCarpetLiquidity,
		minUsableTick,
		minTick,
		true,
	)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount1_.Cmp(leftCarpetCumulativeAmount1) <= 0 && !leftCarpetLiquidity.IsZero() {
		// Use left carpet
		success, roundedTick := uniformLib.InverseCumulativeAmount1(
			tickSpacing,
			cumulativeAmount1_,
			leftCarpetLiquidity,
			minUsableTick,
			minTick,
			true,
		)
		return success, roundedTick, nil
	} else {
		var remainder uint256.Int
		remainder.Sub(cumulativeAmount1_, leftCarpetCumulativeAmount1)
		mainCumulativeAmount1, err := geoLib.CumulativeAmount1(
			tickSpacing,
			minTick+length*tickSpacing,
			mainLiquidity,
			minTick,
			length,
			alphaX96,
		)
		if err != nil {
			return false, 0, err
		}

		if remainder.Cmp(mainCumulativeAmount1) <= 0 {
			// Use main
			success, roundedTick, err := geoLib.InverseCumulativeAmount1(
				tickSpacing,
				&remainder,
				mainLiquidity,
				minTick,
				length,
				alphaX96,
			)
			return success, roundedTick, err
		} else if !rightCarpetLiquidity.IsZero() {
			// Use right carpet
			remainder.Sub(&remainder, mainCumulativeAmount1)
			success, roundedTick := uniformLib.InverseCumulativeAmount1(
				tickSpacing,
				&remainder,
				rightCarpetLiquidity,
				minTick+length*tickSpacing,
				maxUsableTick,
				true,
			)
			return success, roundedTick, nil
		}
	}
	return false, 0, nil
}
