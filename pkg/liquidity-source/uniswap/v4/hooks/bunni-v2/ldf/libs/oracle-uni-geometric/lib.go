package oracleunigeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/geometric"
	uniformLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/uniform"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"

	"github.com/holiman/uint256"
)

type DistributionType int

const (
	Uniform DistributionType = iota
	Geometric
)

// DecodeParams decodes the LDF parameters from bytes32
func DecodeParams(
	tickSpacing int,
	oracleTick int,
	ldfParams [32]byte,
) (
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType DistributionType,
) {
	// | shiftMode - 1 byte | distributionType - 1 byte | oracleIsTickLower - 1 byte | oracleTickOffset - 2 bytes | nonOracleTick - 3 bytes | alpha - 4 bytes |

	distributionType = DistributionType(ldfParams[1])
	oracleIsTickLower := ldfParams[2] != 0

	oracleTickOffsetRaw := int16(uint16(ldfParams[3])<<8 | uint16(ldfParams[4]))
	oracleTickOffset := int(oracleTickOffsetRaw)

	nonOracleTickRaw := uint32(ldfParams[5])<<16 | uint32(ldfParams[6])<<8 | uint32(ldfParams[7])
	nonOracleTick := int(signExtend24to32(nonOracleTickRaw))

	alpha := uint32(ldfParams[8])<<24 | uint32(ldfParams[9])<<16 | uint32(ldfParams[10])<<8 | uint32(ldfParams[11])

	oracleTick += oracleTickOffset

	if oracleIsTickLower {
		tickLower = oracleTick
		tickUpper = nonOracleTick
	} else {
		tickLower = nonOracleTick
		tickUpper = oracleTick
	}

	minUsableTick := math.MinUsableTick(tickSpacing)
	maxUsableTick := math.MaxUsableTick(tickSpacing)

	tickLower = max(minUsableTick, tickLower)
	tickUpper = min(maxUsableTick, tickUpper)

	if tickLower >= tickUpper {
		if oracleIsTickLower {
			tickLower = tickUpper - tickSpacing
		} else {
			tickUpper = tickLower + tickSpacing
		}
	}

	alphaX96 = math.MulDiv(uint256.NewInt(uint64(alpha)), math.Q96, math.ALPHA_BASE)

	return
}

// signExtend24to32 extends a 24-bit signed integer to 32 bits
func signExtend24to32(val uint32) uint32 {
	if val&0x800000 != 0 {
		return val | 0xFF000000
	}
	return val & 0x00FFFFFF
}

// CumulativeAmount0 computes the cumulative amount of token0
func CumulativeAmount0(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType DistributionType,
) (*uint256.Int, error) {
	if distributionType == Uniform {
		return uniformLib.CumulativeAmount0(
			tickSpacing,
			roundedTick,
			totalLiquidity,
			tickLower,
			tickUpper,
			false,
		)
	}

	length := (tickUpper - tickLower) / tickSpacing

	return geoLib.CumulativeAmount0(tickSpacing, roundedTick, totalLiquidity, tickLower, length, alphaX96)
}

// CumulativeAmount1 computes the cumulative amount of token1
func CumulativeAmount1(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType DistributionType,
) (*uint256.Int, error) {
	if distributionType == Uniform {
		return uniformLib.CumulativeAmount1(
			tickSpacing,
			roundedTick,
			totalLiquidity,
			tickLower,
			tickUpper,
			false,
		)
	}

	length := (tickUpper - tickLower) / tickSpacing

	return geoLib.CumulativeAmount1(tickSpacing, roundedTick, totalLiquidity, tickLower, length, alphaX96)
}

// inverseCumulativeAmount0 computes the inverse cumulative amount0
func InverseCumulativeAmount0(
	tickSpacing int,
	cumulativeAmount0_,
	totalLiquidity *uint256.Int,
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType DistributionType,
) (bool, int, error) {
	if distributionType == Uniform {
		success, roundedTick := uniformLib.InverseCumulativeAmount0(
			tickSpacing,
			cumulativeAmount0_,
			totalLiquidity,
			tickLower,
			tickUpper,
			false,
		)

		return success, roundedTick, nil
	}

	length := (tickUpper - tickLower) / tickSpacing

	return geoLib.InverseCumulativeAmount0(tickSpacing, cumulativeAmount0_, totalLiquidity, tickLower, length, alphaX96)
}

// InverseCumulativeAmount1 computes the inverse cumulative amount1
func InverseCumulativeAmount1(
	tickSpacing int,
	cumulativeAmount1_,
	totalLiquidity *uint256.Int,
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType DistributionType,
) (bool, int, error) {
	if distributionType == Uniform {
		success, roundedTick := uniformLib.InverseCumulativeAmount1(
			tickSpacing,
			cumulativeAmount1_,
			totalLiquidity,
			tickLower,
			tickUpper,
			false,
		)

		return success, roundedTick, nil
	}

	length := (tickUpper - tickLower) / tickSpacing

	return geoLib.InverseCumulativeAmount1(tickSpacing, cumulativeAmount1_, totalLiquidity, tickLower, length, alphaX96)
}

// LiquidityDensityX96 computes the liquidity density
func LiquidityDensityX96(
	tickSpacing,
	roundedTick int,
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType DistributionType,
) (*uint256.Int, error) {
	if distributionType == Uniform {
		return uniformLib.LiquidityDensityX96(tickSpacing, roundedTick, tickLower, tickUpper), nil
	}

	length := (tickUpper - tickLower) / tickSpacing

	return geoLib.LiquidityDensityX96(tickSpacing, roundedTick, tickLower, length, alphaX96)
}
