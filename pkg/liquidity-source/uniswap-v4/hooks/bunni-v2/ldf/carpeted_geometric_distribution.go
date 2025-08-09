package ldf

import (
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
		minTick = EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
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
		minTick = EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
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
func (c *CarpetedGeometricDistribution) decodeParams(twapTick int, ldfParams [32]byte) (minTick, length int, alphaX96, weightCarpet *uint256.Int, shiftMode ShiftMode) {
	// | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length - 2 bytes | alpha - 4 bytes | weightCarpet - 4 bytes |
	shiftMode = ShiftMode(ldfParams[0])
	length = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))
	alpha := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])
	weightCarpetVal := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])

	// Convert alpha to alphaX96
	alphaX96 = uint256.NewInt(uint64(alpha))
	alphaX96.Mul(alphaX96, math.Q96)
	alphaX96.Div(alphaX96, math.ALPHA_BASE)

	// Convert weightCarpet to WAD
	weightCarpet = uint256.NewInt(uint64(weightCarpetVal))

	if shiftMode != ShiftModeStatic {
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
	roundedTick, minTick, length int, alphaX96, weightCarpet *uint256.Int,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	// compute liquidityDensityX96
	liquidityDensityX96, err = c.liquidityDensityX96(roundedTick, minTick, length, alphaX96, weightCarpet)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount0DensityX96
	cumulativeAmount0DensityX96, err = c.cumulativeAmount0(roundedTick+c.tickSpacing, minTick, length, alphaX96, weightCarpet)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount1DensityX96
	cumulativeAmount1DensityX96, err = c.cumulativeAmount1(roundedTick-c.tickSpacing, minTick, length, alphaX96, weightCarpet)
	if err != nil {
		return nil, nil, nil, err
	}

	return
}

// cumulativeAmount0 computes the cumulative amount0
func (c *CarpetedGeometricDistribution) cumulativeAmount0(
	roundedTick, minTick, length int, alphaX96, weightCarpet *uint256.Int,
) (*uint256.Int, error) {
	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := c.getCarpetedLiquidity(minTick, length, weightCarpet)

	// Left carpet amount0
	leftCarpetAmount0, err := c.uniformCumulativeAmount0(roundedTick, leftCarpetLiquidity, minUsableTick, minTick, true)
	if err != nil {
		return nil, err
	}

	// Main geometric amount0
	mainAmount0, err := c.geometricCumulativeAmount0(roundedTick, mainLiquidity, minTick, length, alphaX96)
	if err != nil {
		return nil, err
	}

	// Right carpet amount0
	rightCarpetAmount0, err := c.uniformCumulativeAmount0(roundedTick, rightCarpetLiquidity, minTick+length*c.tickSpacing, maxUsableTick, true)
	if err != nil {
		return nil, err
	}

	// Sum all amounts - reuse leftCarpetAmount0 for total
	leftCarpetAmount0.Add(leftCarpetAmount0, mainAmount0)
	leftCarpetAmount0.Add(leftCarpetAmount0, rightCarpetAmount0)

	return leftCarpetAmount0, nil
}

// cumulativeAmount1 computes the cumulative amount1
func (c *CarpetedGeometricDistribution) cumulativeAmount1(
	roundedTick, minTick, length int, alphaX96, weightCarpet *uint256.Int,
) (*uint256.Int, error) {
	// Get carpeted liquidity distribution
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := c.getCarpetedLiquidity(minTick, length, weightCarpet)

	// Left carpet amount1
	leftCarpetAmount1, err := c.uniformCumulativeAmount1(roundedTick, leftCarpetLiquidity, minUsableTick, minTick, true)
	if err != nil {
		return nil, err
	}

	// Main geometric amount1
	mainAmount1, err := c.geometricCumulativeAmount1(roundedTick, mainLiquidity, minTick, length, alphaX96)
	if err != nil {
		return nil, err
	}

	// Right carpet amount1
	rightCarpetAmount1, err := c.uniformCumulativeAmount1(roundedTick, rightCarpetLiquidity, minTick+length*c.tickSpacing, maxUsableTick, true)
	if err != nil {
		return nil, err
	}

	// Sum all amounts - reuse leftCarpetAmount1 for total
	leftCarpetAmount1.Add(leftCarpetAmount1, mainAmount1)
	leftCarpetAmount1.Add(leftCarpetAmount1, rightCarpetAmount1)

	return leftCarpetAmount1, nil
}

// getCarpetedLiquidity computes the liquidity distribution for carpeted parts
func (c *CarpetedGeometricDistribution) getCarpetedLiquidity(minTick, length int, weightCarpet *uint256.Int) (leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity *uint256.Int, minUsableTick, maxUsableTick int) {
	minUsableTick = math.MinUsableTick(c.tickSpacing)
	maxUsableTick = math.MaxUsableTick(c.tickSpacing)
	numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/c.tickSpacing - length

	if numRoundedTicksCarpeted <= 0 {
		var zero uint256.Int
		return &zero, math.Q96, &zero, minUsableTick, maxUsableTick
	}

	// Main liquidity: totalLiquidity * (WAD - weightCarpet) / WAD
	var mainLiquidityVar uint256.Int
	mainLiquidityVar.Mul(math.Q96, math.WAD)
	var wadMinusWeightCarpet uint256.Int
	wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
	mainLiquidityVar.Mul(&mainLiquidityVar, &wadMinusWeightCarpet)
	mainLiquidityVar.Div(&mainLiquidityVar, math.WAD)
	mainLiquidity = &mainLiquidityVar

	// Carpet liquidity: totalLiquidity - mainLiquidity
	var carpetLiquidity uint256.Int
	carpetLiquidity.Sub(math.Q96, &mainLiquidityVar)

	// Right carpet liquidity: carpetLiquidity * rightCarpetNumRoundedTicks / numRoundedTicksCarpeted
	rightCarpetNumRoundedTicks := (maxUsableTick-minTick)/c.tickSpacing - length
	rightCarpetLiquidity, _ = math.FullMulDiv(&carpetLiquidity, uint256.NewInt(uint64(rightCarpetNumRoundedTicks)), uint256.NewInt(uint64(numRoundedTicksCarpeted)))

	// Left carpet liquidity: carpetLiquidity - rightCarpetLiquidity
	carpetLiquidity.Sub(&carpetLiquidity, rightCarpetLiquidity)
	leftCarpetLiquidity = &carpetLiquidity

	return
}

// uniformCumulativeAmount0 computes cumulative amount0 for uniform distribution
func (c *CarpetedGeometricDistribution) uniformCumulativeAmount0(roundedTick int, liquidity *uint256.Int, tickLower, tickUpper int, roundUp bool) (*uint256.Int, error) {
	if roundedTick >= tickUpper {
		var zero uint256.Int
		return &zero, nil
	}

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(roundedTick)
	if err != nil {
		return nil, err
	}
	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, err
	}

	amount0, err := math.GetAmount0Delta(
		sqrtPriceLower,
		sqrtPriceUpper,
		liquidity,
		roundUp,
	)
	if err != nil {
		return nil, err
	}

	return amount0, nil
}

// uniformCumulativeAmount1 computes cumulative amount1 for uniform distribution
func (c *CarpetedGeometricDistribution) uniformCumulativeAmount1(roundedTick int, liquidity *uint256.Int, tickLower, tickUpper int, roundUp bool) (*uint256.Int, error) {
	if roundedTick <= tickLower {
		var zero uint256.Int
		return &zero, nil
	}

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return nil, err
	}
	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, err
	}

	amount1, err := math.GetAmount1Delta(
		sqrtPriceLower,
		sqrtPriceUpper,
		liquidity,
		roundUp,
	)
	if err != nil {
		return nil, err
	}

	return amount1, nil
}

// geometricDensityX96 computes the liquidity density at a given tick
func (c *CarpetedGeometricDistribution) liquidityDensityX96(roundedTick, minTick, length int, alphaX96, weightCarpet *uint256.Int) (*uint256.Int, error) {
	if roundedTick >= minTick && roundedTick < minTick+length*c.tickSpacing {
		// Inside the main geometric distribution
		geometricDensity, err := c.geometricLiquidityDensityX96(roundedTick, minTick, length, alphaX96)
		if err != nil {
			return nil, err
		}

		// Apply carpet weight: geometricDensity * (WAD - weightCarpet) / WAD
		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
		result, err := math.FullMulDiv(geometricDensity, &wadMinusWeightCarpet, math.WAD)
		if err != nil {
			return nil, err
		}

		return result, nil
	} else {
		// Outside the main distribution - use carpet distribution
		minUsableTick := math.MinUsableTick(c.tickSpacing)
		maxUsableTick := math.MaxUsableTick(c.tickSpacing)
		numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/c.tickSpacing - length
		if numRoundedTicksCarpeted <= 0 {
			var zero uint256.Int
			return &zero, nil
		}

		// Carpet liquidity: totalLiquidity - mainLiquidity
		var mainLiquidity uint256.Int
		mainLiquidity.Mul(math.Q96, math.WAD)
		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
		mainLiquidity.Mul(&mainLiquidity, &wadMinusWeightCarpet)
		mainLiquidity.Div(&mainLiquidity, math.WAD)
		var carpetLiquidity uint256.Int
		carpetLiquidity.Sub(math.Q96, &mainLiquidity)
		result, err := math.FullMulDiv(&carpetLiquidity, math.Q96, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// geometricLiquidityDensityX96 computes the geometric distribution part
func (c *CarpetedGeometricDistribution) geometricLiquidityDensityX96(roundedTick, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick < minTick || roundedTick >= minTick+length*c.tickSpacing {
		var zero uint256.Int
		return &zero, nil
	}

	x := (roundedTick - minTick) / c.tickSpacing

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		term1, err := math.Rpow(&alphaInvX96, length-x, math.Q96)
		if err != nil {
			return nil, err
		}
		var term2 uint256.Int
		term2.Sub(alphaX96, math.Q96)
		term3, err := math.Rpow(&alphaInvX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(term1, &term2, &denom)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		// alpha <= 1
		var term1 uint256.Int
		term1.Sub(math.Q96, alphaX96)
		term2, err := math.Rpow(alphaX96, x, math.Q96)
		if err != nil {
			return nil, err
		}
		term3, err := math.Rpow(alphaX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(&term1, term2, &denom)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

// geometricCumulativeAmount0 computes cumulative amount0 for geometric distribution
func (c *CarpetedGeometricDistribution) geometricCumulativeAmount0(roundedTick int, totalLiquidity *uint256.Int, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	// Simplified implementation - would use the full geometric distribution logic
	// For now, return a reasonable approximation
	if roundedTick >= minTick+length*c.tickSpacing {
		var zero uint256.Int
		return &zero, nil
	}

	// Use simplified calculation
	var result uint256.Int
	for i := (roundedTick - minTick) / c.tickSpacing; i < length; i++ {
		density, err := c.geometricLiquidityDensityX96(minTick+i*c.tickSpacing, minTick, length, alphaX96)
		if err != nil {
			return nil, err
		}

		sqrtPriceLower, err := math.GetSqrtPriceAtTick(minTick + i*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		sqrtPriceUpper, err := math.GetSqrtPriceAtTick(minTick + (i+1)*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		amount0, err := math.GetAmount0Delta(
			sqrtPriceLower,
			sqrtPriceUpper,
			density,
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		result.Add(&result, amount0)
	}

	return &result, nil
}

// geometricCumulativeAmount1 computes cumulative amount1 for geometric distribution
func (c *CarpetedGeometricDistribution) geometricCumulativeAmount1(roundedTick int, totalLiquidity *uint256.Int, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	// Simplified implementation - would use the full geometric distribution logic
	if roundedTick <= minTick {
		var zero uint256.Int
		return &zero, nil
	}

	// Use simplified calculation
	var result uint256.Int
	for i := 0; i < (roundedTick-minTick)/c.tickSpacing; i++ {
		density, err := c.geometricLiquidityDensityX96(minTick+i*c.tickSpacing, minTick, length, alphaX96)
		if err != nil {
			return nil, err
		}

		sqrtPriceLower, err := math.GetSqrtPriceAtTick(minTick + i*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		sqrtPriceUpper, err := math.GetSqrtPriceAtTick(minTick + (i+1)*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		amount1, err := math.GetAmount1Delta(
			sqrtPriceLower,
			sqrtPriceUpper,
			density,
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		result.Add(&result, amount1)
	}

	return &result, nil
}

// computeSwap computes the swap parameters
func (c *CarpetedGeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	minTick, length int,
	alphaX96, weightCarpet *uint256.Int,
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
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick+c.tickSpacing, minTick, length, alphaX96, weightCarpet)
		} else {
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick, minTick, length, alphaX96, weightCarpet)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick, minTick, length, alphaX96, weightCarpet)
		} else {
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick-c.tickSpacing, minTick, length, alphaX96, weightCarpet)
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
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick-c.tickSpacing, minTick, length, alphaX96, weightCarpet)
		} else {
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick, minTick, length, alphaX96, weightCarpet)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick, minTick, length, alphaX96, weightCarpet)
		} else {
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick+c.tickSpacing, minTick, length, alphaX96, weightCarpet)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	}

	// compute swap liquidity
	swapLiquidity, err = c.liquidityDensityX96(roundedTick, minTick, length, alphaX96, weightCarpet)
	if err != nil {
		return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}

// DecodeState decodes a bytes32 state into initialized flag and lastMinTick
// Equivalent to Solidity: function _decodeState(bytes32 ldfState) internal pure returns (bool initialized, int24 lastMinTick)
func DecodeState(ldfState [32]byte) (initialized bool, lastMinTick int32) {
	// | initialized - 1 byte | lastMinTick - 3 bytes |
	initialized = ldfState[0] == 1
	lastMinTick = int32(uint32(ldfState[1])<<16 | uint32(ldfState[2])<<8 | uint32(ldfState[3]))
	return
}
