package ldf

import (
	carpetedgeoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/carpeted-geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/shift-mode"

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
	minTick, length, alphaX96, weightCarpet, shiftMode := c.decodeParams(twapTick, ldfParams)
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

	newLdfState = c.encodeState(minTick)
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
	minTick, length, alphaX96, weightCarpet, shiftMode := c.decodeParams(twapTick, ldfParams)
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

// decodeParams decodes the LDF parameters from bytes32
func (c *CarpetedGeometricDistribution) decodeParams(
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
	shiftMode = shiftmode.ShiftMode(ldfParams[0])
	length = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))
	alpha := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])
	weightCarpetVal := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])

	// Convert alpha to alphaX96
	alphaX96 = uint256.NewInt(uint64(alpha))
	alphaX96.Mul(alphaX96, math.Q96)
	alphaX96.Div(alphaX96, math.ALPHA_BASE)

	// Convert weightCarpet to WAD
	weightCarpet = uint256.NewInt(uint64(weightCarpetVal))

	if shiftMode != shiftmode.Static {
		// use rounded TWAP value + offset as minTick
		offset := int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
		minTick = math.RoundTickSingle(twapTick+offset, c.tickSpacing)

		// bound distribution to be within the range of usable ticks
		minUsableTick := math.MinUsableTick(c.tickSpacing)
		maxUsableTick := math.MaxUsableTick(c.tickSpacing)
		if minTick < minUsableTick {
			minTick = minUsableTick
		} else if minTick > maxUsableTick-length*c.tickSpacing {
			minTick = maxUsableTick - length*c.tickSpacing
		}
	} else {
		// static minTick set in params
		minTick = int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
	}

	return
}

// encodeState encodes the state into bytes32
func (c *CarpetedGeometricDistribution) encodeState(minTick int) [32]byte {
	var state [32]byte
	state[0] = 1 // initialized = true
	state[1] = byte((minTick >> 16) & 0xFF)
	state[2] = byte((minTick >> 8) & 0xFF)
	state[3] = byte(minTick & 0xFF)
	return state
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
	// compute liquidityDensityX96
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

	// compute cumulativeAmount0DensityX96
	cumulativeAmount0DensityX96, err = carpetedgeoLib.CumulativeAmount0(
		c.tickSpacing,
		roundedTick+c.tickSpacing,
		minTick,
		length,
		alphaX96,
		weightCarpet,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount1DensityX96
	cumulativeAmount1DensityX96, err = carpetedgeoLib.CumulativeAmount1(
		c.tickSpacing,
		roundedTick-c.tickSpacing,
		minTick,
		length,
		alphaX96,
		weightCarpet,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return
}

// computeSwap computes the swap parameters
func (c *CarpetedGeometricDistribution) computeSwap(
	_,
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
		// compute roundedTick by inverting the cumulative amount0
		// Simplified implementation - would need proper inverse calculation
		roundedTick = minTick + (length/2)*c.tickSpacing

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount0_, err = carpetedgeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick+c.tickSpacing,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount0_, err = carpetedgeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick,
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
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount1_, err = carpetedgeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick-c.tickSpacing,
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
		// compute roundedTick by inverting the cumulative amount1
		// Simplified implementation - would need proper inverse calculation
		roundedTick = minTick + (length/2)*c.tickSpacing

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount1_, err = carpetedgeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick-c.tickSpacing,
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount1_, err = carpetedgeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick,
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
				minTick,
				length,
				alphaX96,
				weightCarpet,
			)
		} else {
			cumulativeAmount0_, err = carpetedgeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick+c.tickSpacing,
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

	// compute swap liquidity
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
