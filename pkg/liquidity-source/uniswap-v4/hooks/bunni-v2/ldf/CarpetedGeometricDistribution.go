package ldf

import (
	carpetedgeoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/carpeted-geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/shift-mode"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// CarpetedGeometricDistribution represents a carpeted geometric distribution LDF
type CarpetedGeometricDistribution struct {
	tickSpacing int
}

// NewCarpetedGeometricDistribution creates a new CarpetedGeometricDistribution
func NewCarpetedGeometricDistribution(tickSpacing int) ILiquidityDensityFunction {
	return &CarpetedGeometricDistribution{
		tickSpacing: tickSpacing,
	}
}

// Query implements the Query method for CarpetedGeometricDistribution
func (c *CarpetedGeometricDistribution) Query(
	roundedTick,
	twapTick,
	spotPriceTick int,
	ldfParams,
	ldfState [32]byte,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	newLdfState [32]byte,
	shouldSurge bool,
	err error,
) {
	minTick, length, alphaX96, weightCarpet, shiftMode := carpetedgeoLib.DecodeParams(c.tickSpacing, twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		minTick = shiftmode.EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
		shouldSurge = minTick != int(lastMinTick)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = c.query(
		roundedTick, minTick, length, alphaX96, weightCarpet,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = EncodeState(minTick)
	return
}

// ComputeSwap implements the ComputeSwap method for CarpetedGeometricDistribution
func (c *CarpetedGeometricDistribution) ComputeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	twapTick,
	_ int,
	ldfParams,
	ldfState [32]byte,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	minTick, length, alphaX96, weightCarpet, shiftMode := carpetedgeoLib.DecodeParams(c.tickSpacing, twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		minTick = shiftmode.EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
	}

	return c.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		minTick,
		length,
		alphaX96,
		weightCarpet,
	)
}

// query computes the liquidity density and cumulative amounts
func (c *CarpetedGeometricDistribution) query(
	roundedTick,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	liquidityDensityX96, err = carpetedgeoLib.LiquidityDensityX96(
		c.tickSpacing,
		roundedTick,
		minTick,
		length,
		alphaX96,
		weightCarpet,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	cumulativeAmount0DensityX96, err = carpetedgeoLib.CumulativeAmount0(
		c.tickSpacing,
		roundedTick+c.tickSpacing,
		math.SCALED_Q96,
		minTick,
		length,
		alphaX96,
		weightCarpet,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	cumulativeAmount0DensityX96.Rsh(cumulativeAmount0DensityX96, QUERY_SCALE_SHIFT)

	cumulativeAmount1DensityX96, err = carpetedgeoLib.CumulativeAmount1(
		c.tickSpacing,
		roundedTick-c.tickSpacing,
		math.SCALED_Q96,
		minTick,
		length,
		alphaX96,
		weightCarpet,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	cumulativeAmount1DensityX96.Rsh(cumulativeAmount1DensityX96, QUERY_SCALE_SHIFT)

	return
}

// computeSwap computes the swap parameters
func (c *CarpetedGeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	minTick,
	length int,
	alphaX96,
	weightCarpet *uint256.Int,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	if exactIn == zeroForOne {
		success, roundedTick, err = carpetedgeoLib.InverseCumulativeAmount0(
			c.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			minTick,
			length,
			alphaX96,
			weightCarpet,
		)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		if exactIn {
			cumulativeAmount0_, err = carpetedgeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick+c.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount0_, err = carpetedgeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount1_, err = carpetedgeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount1_, err = carpetedgeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick-c.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	} else {
		success, roundedTick, err = carpetedgeoLib.InverseCumulativeAmount1(
			c.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			minTick,
			length,
			alphaX96,
			weightCarpet,
		)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		if exactIn {
			cumulativeAmount1_, err = carpetedgeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick-c.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount1_, err = carpetedgeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount0_, err = carpetedgeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount0_, err = carpetedgeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick+c.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	}

	swapLiquidity, err = carpetedgeoLib.LiquidityDensityX96(
		c.tickSpacing,
		roundedTick,
		minTick,
		length,
		alphaX96,
		weightCarpet,
	)
	if err != nil {
		return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
