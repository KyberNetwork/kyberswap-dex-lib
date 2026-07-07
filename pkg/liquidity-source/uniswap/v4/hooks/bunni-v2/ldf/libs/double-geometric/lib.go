package doublegeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// DecodeParams decodes the LDF parameters from bytes32
func DecodeParams(
	tickSpacing,
	twapTick int,
	ldfParams [32]byte,
) (
	minTick,
	length0,
	length1 int,
	alpha0X96,
	alpha1X96,
	weight0,
	weight1 *uint256.Int,
	shiftMode shiftmode.ShiftMode,
) {
	// | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes |
	shiftMode = shiftmode.ShiftMode(ldfParams[0])

	length0 = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))

	alpha0 := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])

	weight0Val := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])

	length1 = int(int16(uint16(ldfParams[14])<<8 | uint16(ldfParams[15])))

	alpha1 := uint32(ldfParams[16])<<24 | uint32(ldfParams[17])<<16 | uint32(ldfParams[18])<<8 | uint32(ldfParams[19])

	weight1Val := uint32(ldfParams[20])<<24 | uint32(ldfParams[21])<<16 | uint32(ldfParams[22])<<8 | uint32(ldfParams[23])

	alpha0X96 = uint256.NewInt(uint64(alpha0))
	alpha0X96.Mul(alpha0X96, math.Q96)
	alpha0X96.Div(alpha0X96, math.ALPHA_BASE)

	alpha1X96 = uint256.NewInt(uint64(alpha1))
	alpha1X96.Mul(alpha1X96, math.Q96)
	alpha1X96.Div(alpha1X96, math.ALPHA_BASE)

	weight0 = uint256.NewInt(uint64(weight0Val))
	weight1 = uint256.NewInt(uint64(weight1Val))

	if shiftMode != shiftmode.Static {
		offsetRaw := uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])

		if offsetRaw&0x800000 != 0 {
			offsetRaw |= 0xFF000000
		}
		offset := int(int32(offsetRaw))

		minTick = math.RoundTickSingle(twapTick+offset, tickSpacing)

		minUsableTick := math.MinUsableTick(tickSpacing)
		maxUsableTick := math.MaxUsableTick(tickSpacing)
		if minTick < minUsableTick {
			minTick = minUsableTick
		} else if minTick > maxUsableTick-(length0+length1)*tickSpacing {
			minTick = maxUsableTick - (length0+length1)*tickSpacing
		}
	} else {
		minTickRaw := uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])

		if minTickRaw&0x800000 != 0 {
			minTickRaw |= 0xFF000000
		}
		minTick = int(int32(minTickRaw))
	}

	return
}

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
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity0 uint256.Int
	totalLiquidity0.MulDivOverflow(totalLiquidity, weight0, &totalWeight)

	var totalLiquidity1 uint256.Int
	totalLiquidity1.MulDivOverflow(totalLiquidity, weight1, &totalWeight)

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

	return amount0_0.Add(amount0_0, amount0_1), nil
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
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity0 uint256.Int
	totalLiquidity0.MulDivOverflow(totalLiquidity, weight0, &totalWeight)

	var totalLiquidity1 uint256.Int
	totalLiquidity1.MulDivOverflow(totalLiquidity, weight1, &totalWeight)

	amount1_0, err := geoLib.CumulativeAmount1(tickSpacing, roundedTick, &totalLiquidity0, minTick+length1*tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	amount1_1, err := geoLib.CumulativeAmount1(tickSpacing, roundedTick, &totalLiquidity1, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	return amount1_0.Add(amount1_0, amount1_1), nil
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
	totalLiquidity0.MulDivOverflow(totalLiquidity, weight0, &totalWeight)

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
		totalLiquidity1.MulDivOverflow(totalLiquidity, weight1, &totalWeight)

		return geoLib.InverseCumulativeAmount0(tickSpacing, &remainder, &totalLiquidity1, minTick, length1, alpha1X96)
	}
}

// inverseCumulativeAmount1 computes the inverse cumulative amount1
func InverseCumulativeAmount1(tickSpacing int, cumulativeAmount1_, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (bool, int, error) {
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)

	var totalLiquidity1 uint256.Int
	totalLiquidity1.MulDivOverflow(totalLiquidity, weight1, &totalWeight)

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
		totalLiquidity0.MulDivOverflow(totalLiquidity, weight0, &totalWeight)

		return geoLib.InverseCumulativeAmount1(tickSpacing, &remainder, &totalLiquidity0, minTick+length1*tickSpacing, length0, alpha0X96)
	}
}

// LiquidityDensityX96 computes the liquidity density using weighted sum of two geometric distributions
func LiquidityDensityX96(tickSpacing, roundedTick, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (*uint256.Int, error) {
	density0, err := geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick+length1*tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	density1, err := geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	result := math.WeightedSum(density0, weight0, density1, weight1)
	return result, nil
}
