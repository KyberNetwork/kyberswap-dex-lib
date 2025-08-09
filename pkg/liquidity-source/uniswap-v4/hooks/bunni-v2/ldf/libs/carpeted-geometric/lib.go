package carpetedgeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/geometric"
	uniformLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/uniform"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// cumulativeAmount0 computes the cumulative amount0
func CumulativeAmount0(
	tickSpacing,
	roundedTick,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (*uint256.Int, error) {
	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := getCarpetedLiquidity(
		tickSpacing,
		minTick,
		length,
		weightCarpet,
	)

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

	// Sum all amounts - reuse leftCarpetAmount0 for total
	leftCarpetAmount0.Add(leftCarpetAmount0, mainAmount0)
	leftCarpetAmount0.Add(leftCarpetAmount0, rightCarpetAmount0)

	return leftCarpetAmount0, nil
}

// CumulativeAmount1 computes the cumulative amount1
func CumulativeAmount1(
	tickSpacing,
	roundedTick,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (*uint256.Int, error) {
	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := getCarpetedLiquidity(
		tickSpacing,
		minTick,
		length,
		weightCarpet,
	)

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
	mainAmount1, err := geoLib.CumulativeAmount1(tickSpacing,
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

	// Sum all amounts - reuse leftCarpetAmount1 for total
	leftCarpetAmount1.Add(leftCarpetAmount1, mainAmount1)
	leftCarpetAmount1.Add(leftCarpetAmount1, rightCarpetAmount1)

	return leftCarpetAmount1, nil
}

// getCarpetedLiquidity computes the liquidity distribution for carpeted parts
func getCarpetedLiquidity(
	tickSpacing,
	minTick,
	length int,
	weightCarpet *uint256.Int,
) (
	leftCarpetLiquidity,
	mainLiquidity,
	rightCarpetLiquidity *uint256.Int,
	minUsableTick,
	maxUsableTick int,
) {
	minUsableTick = math.MinUsableTick(tickSpacing)
	maxUsableTick = math.MaxUsableTick(tickSpacing)
	numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/tickSpacing - length

	if numRoundedTicksCarpeted <= 0 {
		var zero uint256.Int
		return &zero, math.Q96, &zero, minUsableTick, maxUsableTick
	}

	// Main liquidity: totalLiquidity * (WAD - weightCarpet) / WAD
	var mainLiquidityVar uint256.Int
	mainLiquidityVar.Mul(math.Q96, math.WAD)
	var wadMinusWeightCarpet uint256.Int
	wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
	mainLiquidityVar.Mul(&mainLiquidityVar, &wadMinusWeightCarpet)
	mainLiquidityVar.Div(&mainLiquidityVar, math.WAD)
	mainLiquidity = &mainLiquidityVar

	// Carpet liquidity: totalLiquidity - mainLiquidity
	var carpetLiquidity uint256.Int
	carpetLiquidity.Sub(math.Q96, &mainLiquidityVar)

	// Right carpet liquidity: carpetLiquidity * rightCarpetNumRoundedTicks / numRoundedTicksCarpeted
	rightCarpetNumRoundedTicks := (maxUsableTick-minTick)/tickSpacing - length
	rightCarpetLiquidity, _ = math.FullMulDiv(
		&carpetLiquidity,
		uint256.NewInt(uint64(rightCarpetNumRoundedTicks)),
		uint256.NewInt(uint64(numRoundedTicksCarpeted)),
	)

	// Left carpet liquidity: carpetLiquidity - rightCarpetLiquidity
	carpetLiquidity.Sub(&carpetLiquidity, rightCarpetLiquidity)
	leftCarpetLiquidity = &carpetLiquidity

	return
}

// geometricDensityX96 computes the liquidity density at a given tick
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
			var zero uint256.Int
			return &zero, nil
		}

		// Carpet liquidity: totalLiquidity - mainLiquidity
		var mainLiquidity uint256.Int
		mainLiquidity.Mul(math.Q96, math.WAD)
		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
		mainLiquidity.Mul(&mainLiquidity, &wadMinusWeightCarpet)
		mainLiquidity.Div(&mainLiquidity, math.WAD)
		var carpetLiquidity uint256.Int
		carpetLiquidity.Sub(math.Q96, &mainLiquidity)
		result, err := math.FullMulDiv(&carpetLiquidity, math.Q96, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}
