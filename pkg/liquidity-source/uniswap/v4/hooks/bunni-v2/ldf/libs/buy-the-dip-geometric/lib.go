package buythedipgeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/geometric"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// DecodeParams decodes the LDF parameters from bytes32
func DecodeParams(ldfParams [32]byte) (
	minTick,
	length,
	altThreshold int,
	alphaX96,
	altAlphaX96 *uint256.Int,
	altThresholdDirection bool,
) {
	// | shiftMode - 1 byte | minTick - 3 bytes | length - 2 bytes | alpha - 4 bytes | altAlpha - 4 bytes | altThreshold - 3 bytes | altThresholdDirection - 1 byte |
	minTickRaw := uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])
	if minTickRaw&0x800000 != 0 {
		minTickRaw |= 0xFF000000
	}
	minTick = int(int32(minTickRaw))

	length = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))

	alpha := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])
	alphaX96 = uint256.NewInt(uint64(alpha))
	alphaX96.Mul(alphaX96, math.Q96)
	alphaX96.Div(alphaX96, math.ALPHA_BASE)

	altAlpha := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])
	altAlphaX96 = uint256.NewInt(uint64(altAlpha))
	altAlphaX96.Mul(altAlphaX96, math.Q96)
	altAlphaX96.Div(altAlphaX96, math.ALPHA_BASE)

	altThresholdRaw := uint32(ldfParams[14])<<16 | uint32(ldfParams[15])<<8 | uint32(ldfParams[16])
	if altThresholdRaw&0x800000 != 0 {
		altThresholdRaw |= 0xFF000000
	}
	altThreshold = int(int32(altThresholdRaw))

	altThresholdDirection = ldfParams[17] != 0

	return
}

func ShouldUseAltAlpha(twapTick, altThreshold int, altThresholdDirection bool) bool {
	if altThresholdDirection {
		return twapTick <= altThreshold
	}
	return twapTick >= altThreshold
}

// Query queries the liquidity density and cumulative amounts at the given rounded tick
func Query(
	roundedTick,
	tickSpacing,
	twapTick,
	minTick,
	length int,
	alphaX96,
	altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	liquidityDensityX96, err = LiquidityDensityX96(
		tickSpacing,
		roundedTick,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	cumulativeAmount0DensityX96, err = CumulativeAmount0(
		tickSpacing,
		roundedTick+tickSpacing,
		math.Q96,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	cumulativeAmount1DensityX96, err = CumulativeAmount1(
		tickSpacing,
		roundedTick-tickSpacing,
		math.Q96,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, nil
}

// liquidityDensityX96 computes the liquidity density at a given tick
func LiquidityDensityX96(
	tickSpacing,
	roundedTick,
	twapTick,
	minTick,
	length int,
	alphaX96,
	altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (*uint256.Int, error) {
	if ShouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) {
		return geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick, length, altAlphaX96)
	}

	return geoLib.LiquidityDensityX96(tickSpacing, roundedTick, minTick, length, alphaX96)
}

// CumulativeAmount0 computes the cumulative amount0
func CumulativeAmount0(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	twapTick,
	minTick,
	length int,
	alphaX96,
	altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (*uint256.Int, error) {
	if ShouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) {
		return geoLib.CumulativeAmount0(tickSpacing, roundedTick, totalLiquidity, minTick, length, altAlphaX96)
	}

	return geoLib.CumulativeAmount0(tickSpacing, roundedTick, totalLiquidity, minTick, length, alphaX96)
}

// CumulativeAmount1 computes the cumulative amount1
func CumulativeAmount1(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	twapTick,
	minTick,
	length int,
	alphaX96,
	altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (*uint256.Int, error) {
	if ShouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) {
		return geoLib.CumulativeAmount1(tickSpacing, roundedTick, totalLiquidity, minTick, length, altAlphaX96)
	}

	return geoLib.CumulativeAmount1(tickSpacing, roundedTick, totalLiquidity, minTick, length, alphaX96)
}

// InverseCumulativeAmount0 computes the inverse of cumulativeAmount0
func InverseCumulativeAmount0(
	tickSpacing int,
	cumulativeAmount0_,
	totalLiquidity *uint256.Int,
	twapTick, minTick, length int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (bool, int, error) {
	if ShouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) {
		return geoLib.InverseCumulativeAmount0(tickSpacing, cumulativeAmount0_, totalLiquidity, minTick, length, altAlphaX96)
	}

	return geoLib.InverseCumulativeAmount0(tickSpacing, cumulativeAmount0_, totalLiquidity, minTick, length, alphaX96)
}

// InverseCumulativeAmount1 computes the inverse of cumulativeAmount1
func InverseCumulativeAmount1(
	tickSpacing int,
	cumulativeAmount0_,
	totalLiquidity *uint256.Int,
	twapTick, minTick, length int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (bool, int, error) {
	if ShouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) {
		return geoLib.InverseCumulativeAmount1(tickSpacing, cumulativeAmount0_, totalLiquidity, minTick, length, altAlphaX96)
	}

	return geoLib.InverseCumulativeAmount1(tickSpacing, cumulativeAmount0_, totalLiquidity, minTick, length, alphaX96)
}
