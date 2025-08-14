package carpeteddoublegeometric

import (
	doubleGeoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/double-geometric"
	uniformLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/uniform"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/shift-mode"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

type Params struct {
	MinTick, Length0, Length1                            int
	Alpha0X96, Alpha1X96, Weight0, Weight1, WeightCarpet *uint256.Int
	ShiftMode                                            shiftmode.ShiftMode
}

// DecodeParams decodes the LDF parameters from bytes32
func DecodeParams(tickSpacing, twapTick int, ldfParams [32]byte) Params {
	// | shiftMode - 1 byte | offset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes | weightCarpet - 4 bytes |

	weightCarpetVal := uint32(ldfParams[24])<<24 | uint32(ldfParams[25])<<16 | uint32(ldfParams[26])<<8 | uint32(ldfParams[27])
	weightCarpet := uint256.NewInt(uint64(weightCarpetVal))

	minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1, shiftMode :=
		doubleGeoLib.DecodeParams(tickSpacing, twapTick, ldfParams)

	return Params{
		MinTick:      minTick,
		Length0:      length0,
		Length1:      length1,
		Alpha0X96:    alpha0X96,
		Alpha1X96:    alpha1X96,
		Weight0:      weight0,
		Weight1:      weight1,
		WeightCarpet: weightCarpet,
		ShiftMode:    shiftMode,
	}
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

		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, params.WeightCarpet)

		return math.MulWad(mainDensity, &wadMinusWeightCarpet)
	} else {
		minUsableTick := math.MinUsableTick(tickSpacing)
		maxUsableTick := math.MaxUsableTick(tickSpacing)
		numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/tickSpacing - length
		if numRoundedTicksCarpeted <= 0 {
			return uint256.NewInt(0), nil
		}

		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, params.WeightCarpet)

		carpetLiquidity, err := math.MulWad(math.Q96, &wadMinusWeightCarpet)
		if err != nil {
			return nil, err
		}

		carpetLiquidity.Sub(math.Q96, carpetLiquidity)

		return math.DivUp(carpetLiquidity, uint256.NewInt(uint64(numRoundedTicksCarpeted))), nil
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

	var wadMinusWeightCarpet uint256.Int
	wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)

	mainLiquidity, err := math.MulWad(totalLiquidity, &wadMinusWeightCarpet)
	if err != nil {
		return nil, nil, nil, 0, 0
	}

	var carpetLiquidity uint256.Int
	carpetLiquidity.Sub(totalLiquidity, mainLiquidity)

	rightCarpetLiquidity, _ = new(uint256.Int).MulDivOverflow(
		&carpetLiquidity,
		uint256.NewInt(uint64((maxUsableTick-minTick)/tickSpacing-length)),
		uint256.NewInt(uint64(numRoundedTicksCarpeted)),
	)

	leftCarpetLiquidity = carpetLiquidity.Sub(&carpetLiquidity, rightCarpetLiquidity)

	return
}
