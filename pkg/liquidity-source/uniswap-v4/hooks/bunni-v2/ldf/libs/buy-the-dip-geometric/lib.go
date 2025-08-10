package buythedipgeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/geometric"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// ShouldUseAltAlpha determines whether to use the alternative alpha value based on TWAP tick and threshold
func ShouldUseAltAlpha(twapTick, altThreshold int, altThresholdDirection bool) bool {
	if altThresholdDirection {
		return twapTick <= altThreshold
	}
	return twapTick >= altThreshold
}

// Query queries the liquidity density and cumulative amounts at the given rounded tick
// This matches the Solidity LibBuyTheDipGeometricDistribution.query function
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
	// compute liquidityDensityX96
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

	// compute cumulativeAmount0DensityX96
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

	// compute cumulativeAmount1DensityX96
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
