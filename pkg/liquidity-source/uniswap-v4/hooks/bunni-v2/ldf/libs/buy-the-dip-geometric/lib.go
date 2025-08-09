package buythedipgeometric

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/geometric"
	"github.com/holiman/uint256"
)

func ShouldUseAltAlpha(twapTick, altThreshold int, altThresholdDirection bool) bool {
	if altThresholdDirection {
		return twapTick <= altThreshold
	}
	return twapTick >= altThreshold
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
