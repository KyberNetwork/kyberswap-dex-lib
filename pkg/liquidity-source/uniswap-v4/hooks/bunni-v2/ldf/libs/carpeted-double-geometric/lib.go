package carpeteddoublegeometric

import (
	doubleGeoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/double-geometric"
	uniformLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/uniform"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/shift-mode"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// Params represents the parameters for CarpetedDoubleGeometricDistribution
type Params struct {
	MinTick, Length0, Length1                            int
	Alpha0X96, Alpha1X96, Weight0, Weight1, WeightCarpet *uint256.Int
	ShiftMode                                            shiftmode.ShiftMode
}

// CumulativeAmount0 computes the cumulative amount of token0
func CumulativeAmount0(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	params Params,
) (*uint256.Int, error) {
	length := params.Length0 + params.Length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		params.MinTick,
		length,
		params.WeightCarpet,
	)

	// Left carpet amount0
	leftCarpetAmount0, err := uniformLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		leftCarpetLiquidity,
		minUsableTick,
		params.MinTick,
		true,
	)
	if err != nil {
		return nil, err
	}

	// Main double geometric amount0
	mainAmount0, err := doubleGeoLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		mainLiquidity,
		params.MinTick,
		params.Length0,
		params.Length1,
		params.Alpha0X96,
		params.Alpha1X96,
		params.Weight0,
		params.Weight1,
	)
	if err != nil {
		return nil, err
	}

	// Right carpet amount0
	rightCarpetAmount0, err := uniformLib.CumulativeAmount0(
		tickSpacing,
		roundedTick,
		rightCarpetLiquidity,
		params.MinTick+length*tickSpacing,
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

// CumulativeAmount1 computes the cumulative amount of token1
func CumulativeAmount1(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	params Params,
) (*uint256.Int, error) {
	length := params.Length0 + params.Length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		params.MinTick,
		length,
		params.WeightCarpet,
	)

	// Left carpet amount1
	leftCarpetAmount1, err := uniformLib.CumulativeAmount1(
		tickSpacing,
		roundedTick,
		leftCarpetLiquidity,
		minUsableTick,
		params.MinTick,
		true,
	)
	if err != nil {
		return nil, err
	}

	// Main double geometric amount1
	mainAmount1, err := doubleGeoLib.CumulativeAmount1(
		tickSpacing,
		roundedTick,
		mainLiquidity,
		params.MinTick,
		params.Length0,
		params.Length1,
		params.Alpha0X96,
		params.Alpha1X96,
		params.Weight0,
		params.Weight1,
	)
	if err != nil {
		return nil, err
	}

	// Right carpet amount1
	rightCarpetAmount1, err := uniformLib.CumulativeAmount1(
		tickSpacing,
		roundedTick,
		rightCarpetLiquidity,
		params.MinTick+length*tickSpacing,
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

// inverseCumulativeAmount0 computes the inverse cumulative amount0
func InverseCumulativeAmount0(
	tickSpacing int,
	cumulativeAmount0_,
	totalLiquidity *uint256.Int,
	params Params,
) (bool, int, error) {
	if cumulativeAmount0_.IsZero() {
		return true, math.MaxUsableTick(tickSpacing), nil
	}

	// try LDFs in the order of right carpet, main, left carpet
	length := params.Length0 + params.Length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		params.MinTick,
		length,
		params.WeightCarpet,
	)

	rightCarpetCumulativeAmount0, err := uniformLib.CumulativeAmount0(
		tickSpacing,
		params.MinTick+length*tickSpacing,
		rightCarpetLiquidity,
		params.MinTick+length*tickSpacing,
		maxUsableTick,
		true,
	)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount0_.Cmp(rightCarpetCumulativeAmount0) <= 0 && !rightCarpetLiquidity.IsZero() {
		// use right carpet
		success, roundedTick := uniformLib.InverseCumulativeAmount0(
			tickSpacing,
			cumulativeAmount0_,
			rightCarpetLiquidity,
			params.MinTick+length*tickSpacing,
			maxUsableTick,
			true,
		)
		return success, roundedTick, nil
	} else {
		// Reuse rightCarpetCumulativeAmount0 for remainder calculation
		rightCarpetCumulativeAmount0.Sub(cumulativeAmount0_, rightCarpetCumulativeAmount0)

		mainCumulativeAmount0, err := doubleGeoLib.CumulativeAmount0(
			tickSpacing,
			params.MinTick,
			mainLiquidity,
			params.MinTick,
			params.Length0,
			params.Length1,
			params.Alpha0X96,
			params.Alpha1X96,
			params.Weight0,
			params.Weight1,
		)
		if err != nil {
			return false, 0, err
		}

		if rightCarpetCumulativeAmount0.Cmp(mainCumulativeAmount0) <= 0 {
			// use main
			return doubleGeoLib.InverseCumulativeAmount0(
				tickSpacing,
				rightCarpetCumulativeAmount0,
				mainLiquidity,
				params.MinTick,
				params.Length0,
				params.Length1,
				params.Alpha0X96,
				params.Alpha1X96,
				params.Weight0,
				params.Weight1,
			)
		} else if !leftCarpetLiquidity.IsZero() {
			// use left carpet - reuse rightCarpetCumulativeAmount0 for final remainder
			rightCarpetCumulativeAmount0.Sub(rightCarpetCumulativeAmount0, mainCumulativeAmount0)
			success, roundedTick := uniformLib.InverseCumulativeAmount0(
				tickSpacing,
				rightCarpetCumulativeAmount0,
				leftCarpetLiquidity,
				minUsableTick,
				params.MinTick,
				true,
			)
			return success, roundedTick, nil
		}
	}
	return false, 0, nil
}

// InverseCumulativeAmount1 computes the inverse cumulative amount1
func InverseCumulativeAmount1(
	tickSpacing int,
	cumulativeAmount1_,
	totalLiquidity *uint256.Int,
	params Params,
) (bool, int, error) {
	if cumulativeAmount1_.IsZero() {
		return true, math.MinUsableTick(tickSpacing) - tickSpacing, nil
	}

	// try LDFs in the order of left carpet, main, right carpet
	length := params.Length0 + params.Length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := getCarpetedLiquidity(
		tickSpacing,
		totalLiquidity,
		params.MinTick,
		length,
		params.WeightCarpet,
	)

	leftCarpetCumulativeAmount1, err := uniformLib.CumulativeAmount1(
		tickSpacing,
		params.MinTick,
		leftCarpetLiquidity,
		minUsableTick,
		params.MinTick,
		true,
	)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount1_.Cmp(leftCarpetCumulativeAmount1) <= 0 && !leftCarpetLiquidity.IsZero() {
		// use left carpet
		success, roundedTick := uniformLib.InverseCumulativeAmount1(
			tickSpacing,
			cumulativeAmount1_,
			leftCarpetLiquidity,
			minUsableTick,
			params.MinTick,
			true,
		)
		return success, roundedTick, nil
	} else {
		// Reuse leftCarpetCumulativeAmount1 for remainder calculation
		leftCarpetCumulativeAmount1.Sub(cumulativeAmount1_, leftCarpetCumulativeAmount1)

		mainCumulativeAmount1, err := doubleGeoLib.CumulativeAmount1(tickSpacing,
			params.MinTick+length*tickSpacing,
			mainLiquidity,
			params.MinTick,
			params.Length0,
			params.Length1,
			params.Alpha0X96,
			params.Alpha1X96,
			params.Weight0,
			params.Weight1,
		)
		if err != nil {
			return false, 0, err
		}

		if leftCarpetCumulativeAmount1.Cmp(mainCumulativeAmount1) <= 0 {
			// use main
			return doubleGeoLib.InverseCumulativeAmount1(
				tickSpacing,
				leftCarpetCumulativeAmount1,
				mainLiquidity,
				params.MinTick,
				params.Length0,
				params.Length1,
				params.Alpha0X96,
				params.Alpha1X96,
				params.Weight0,
				params.Weight1,
			)
		} else if !rightCarpetLiquidity.IsZero() {
			// use right carpet - reuse leftCarpetCumulativeAmount1 for final remainder
			leftCarpetCumulativeAmount1.Sub(leftCarpetCumulativeAmount1, mainCumulativeAmount1)
			success, roundedTick := uniformLib.InverseCumulativeAmount1(tickSpacing,
				leftCarpetCumulativeAmount1,
				rightCarpetLiquidity,
				params.MinTick+length*tickSpacing,
				maxUsableTick,
				true,
			)
			return success, roundedTick, nil
		}
	}
	return false, 0, nil
}

// LiquidityDensityX96 computes the liquidity density
func LiquidityDensityX96(
	tickSpacing,
	roundedTick int,
	params Params,
) (*uint256.Int, error) {
	length := params.Length0 + params.Length1
	if roundedTick >= params.MinTick && roundedTick < params.MinTick+length*tickSpacing {
		// Main distribution
		mainDensity, err := doubleGeoLib.LiquidityDensityX96(
			tickSpacing,
			roundedTick,
			params.MinTick,
			params.Length0,
			params.Length1,
			params.Alpha0X96,
			params.Alpha1X96,
			params.Weight0,
			params.Weight1,
		)
		if err != nil {
			return nil, err
		}

		// Apply carpet weight: mainDensity * (WAD - weightCarpet) / WAD
		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, params.WeightCarpet)
		result, err := math.FullMulDiv(mainDensity, &wadMinusWeightCarpet, math.WAD)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		// Carpet distribution
		minUsableTick := math.MinUsableTick(tickSpacing)
		maxUsableTick := math.MaxUsableTick(tickSpacing)
		numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/tickSpacing - length
		if numRoundedTicksCarpeted <= 0 {
			return uint256.NewInt(0), nil
		}

		// Carpet liquidity: Q96 * weightCarpet / numRoundedTicksCarpeted
		var carpetLiquidity uint256.Int
		carpetLiquidity.Mul(math.Q96, params.WeightCarpet)
		result, err := math.FullMulDiv(&carpetLiquidity, math.Q96, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// getCarpetedLiquidity computes the carpeted liquidity distribution
func getCarpetedLiquidity(
	tickSpacing int,
	totalLiquidity *uint256.Int,
	minTick, length int,
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
		return uint256.NewInt(0), totalLiquidity, uint256.NewInt(0), minUsableTick, maxUsableTick
	}

	// Main liquidity: totalLiquidity * (WAD - weightCarpet) / WAD
	var mainLiquidityVar uint256.Int
	mainLiquidityVar.Mul(totalLiquidity, math.WAD)
	var wadMinusWeightCarpet uint256.Int
	wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
	mainLiquidityVar.Mul(&mainLiquidityVar, &wadMinusWeightCarpet)
	mainLiquidityVar.Div(&mainLiquidityVar, math.WAD)
	mainLiquidity = &mainLiquidityVar

	// Carpet liquidity: totalLiquidity - mainLiquidity
	var carpetLiquidity uint256.Int
	carpetLiquidity.Sub(totalLiquidity, &mainLiquidityVar)

	// Right carpet liquidity: carpetLiquidity * rightCarpetTicks / numRoundedTicksCarpeted
	rightCarpetTicks := (maxUsableTick-minTick)/tickSpacing - length
	var rightCarpetLiquidityVar uint256.Int
	rightCarpetLiquidityVar.Mul(&carpetLiquidity, uint256.NewInt(uint64(rightCarpetTicks)))
	rightCarpetLiquidityVar.Div(&rightCarpetLiquidityVar, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
	rightCarpetLiquidity = &rightCarpetLiquidityVar

	// Left carpet liquidity: carpetLiquidity - rightCarpetLiquidity
	carpetLiquidity.Sub(&carpetLiquidity, &rightCarpetLiquidityVar)
	leftCarpetLiquidity = &carpetLiquidity

	return
}
