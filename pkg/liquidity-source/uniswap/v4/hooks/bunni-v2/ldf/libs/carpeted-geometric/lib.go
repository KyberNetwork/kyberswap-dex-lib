package carpetedgeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/geometric"
	uniformLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/uniform"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

// DecodeParams decodes the LDF parameters from bytes32
func DecodeParams(
	tickSpacing,
	twapTick int,
	ldfParams [32]byte,
) (
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
	shiftMode shiftmode.ShiftMode,
) {
	// | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length - 2 bytes | alpha - 4 bytes | weightCarpet - 4 bytes |
	minTick, length, alphaX96, shiftMode = geoLib.DecodeParams(tickSpacing, twapTick, ldfParams)
	weightCarpetVal := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])
	weightCarpet = uint256.NewInt(uint64(weightCarpetVal))
	return
}

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

	var wadMinusWeightCarpet uint256.Int
	wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
	mainLiquidity, err = math.FullMulDiv(totalLiquidity, &wadMinusWeightCarpet, math.WAD)
	if err != nil {
		return nil, nil, nil, 0, 0, err
	}

	var carpetLiquidity uint256.Int
	carpetLiquidity.Sub(totalLiquidity, mainLiquidity)

	rightCarpetNumRoundedTicks := (maxUsableTick-minTick)/tickSpacing - length
	rightCarpetLiquidity, err = math.FullMulDiv(
		&carpetLiquidity,
		uint256.NewInt(uint64(rightCarpetNumRoundedTicks)),
		uint256.NewInt(uint64(numRoundedTicksCarpeted)),
	)
	if err != nil {
		return nil, nil, nil, 0, 0, err
	}

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
		geometricDensity, err := geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick, length, alphaX96)
		if err != nil {
			return nil, err
		}

		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
		result, err := math.FullMulDiv(geometricDensity, &wadMinusWeightCarpet, math.WAD)
		if err != nil {
			return nil, err
		}

		return result, nil
	} else {
		minUsableTick := math.MinUsableTick(tickSpacing)
		maxUsableTick := math.MaxUsableTick(tickSpacing)
		numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/tickSpacing - length
		if numRoundedTicksCarpeted <= 0 {
			return u256.U0, nil
		}

		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
		mainLiquidity, err := math.FullMulDiv(math.Q96, &wadMinusWeightCarpet, math.WAD)
		if err != nil {
			return nil, err
		}

		var carpetLiquidity uint256.Int
		carpetLiquidity.Sub(math.Q96, mainLiquidity)

		result := math.DivUp(&carpetLiquidity, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
		return result, nil
	}
}

// InverseCumulativeAmount0 computes the inverse of cumulative amount0
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
