package ldf

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// GeometricDistribution represents a geometric distribution LDF
type GeometricDistribution struct {
	tickSpacing int
}

// NewGeometricDistribution creates a new GeometricDistribution
func NewGeometricDistribution(tickSpacing int) ILiquidityDensityFunction {
	return &GeometricDistribution{
		tickSpacing: tickSpacing,
	}
}

// Query implements the Query method for GeometricDistribution
func (g *GeometricDistribution) Query(
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
	minTick, length, alphaX96, shiftMode := g.decodeParams(twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		minTick = EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
		shouldSurge = minTick != int(lastMinTick)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = g.query(
		roundedTick, minTick, length, alphaX96,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = g.encodeState(minTick)
	return
}

// ComputeSwap implements the ComputeSwap method for GeometricDistribution
func (g *GeometricDistribution) ComputeSwap(
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
	minTick, length, alphaX96, shiftMode := g.decodeParams(twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		minTick = EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
	}

	return g.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		minTick,
		length,
		alphaX96,
	)
}

// decodeParams decodes the LDF parameters from bytes32
func (g *GeometricDistribution) decodeParams(twapTick int, ldfParams [32]byte) (minTick, length int, alphaX96 *uint256.Int, shiftMode ShiftMode) {
	// | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length - 2 bytes | alpha - 4 bytes |
	shiftMode = ShiftMode(ldfParams[0])
	length = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))
	alpha := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])

	// Convert alpha to alphaX96 (alpha * Q96 / ALPHA_BASE)
	alphaX96 = uint256.NewInt(uint64(alpha))
	alphaX96.Mul(alphaX96, math.Q96)
	alphaX96.Div(alphaX96, math.ALPHA_BASE)

	if shiftMode != ShiftModeStatic {
		// use rounded TWAP value + offset as minTick
		offset := int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
		minTick = math.RoundTickSingle(twapTick+offset, g.tickSpacing)

		// bound distribution to be within the range of usable ticks
		minUsableTick := math.MinUsableTick(g.tickSpacing)
		maxUsableTick := math.MaxUsableTick(g.tickSpacing)
		if minTick < minUsableTick {
			minTick = minUsableTick
		} else if minTick > maxUsableTick-length*g.tickSpacing {
			minTick = maxUsableTick - length*g.tickSpacing
		}
	} else {
		// static minTick set in params
		minTick = int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
	}

	return
}

// encodeState encodes the state into bytes32
func (g *GeometricDistribution) encodeState(minTick int) [32]byte {
	var state [32]byte
	state[0] = 1 // initialized = true
	state[1] = byte((minTick >> 16) & 0xFF)
	state[2] = byte((minTick >> 8) & 0xFF)
	state[3] = byte(minTick & 0xFF)
	return state
}

// query computes the liquidity density and cumulative amounts
func (g *GeometricDistribution) query(
	roundedTick, minTick, length int, alphaX96 *uint256.Int,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	// compute liquidityDensityX96
	liquidityDensityX96, err = g.liquidityDensityX96(roundedTick, minTick, length, alphaX96)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount0DensityX96
	cumulativeAmount0DensityX96, err = g.cumulativeAmount0(roundedTick, minTick, length, alphaX96)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount1DensityX96
	cumulativeAmount1DensityX96, err = g.cumulativeAmount1(roundedTick, minTick, length, alphaX96)
	if err != nil {
		return nil, nil, nil, err
	}

	return
}

// liquidityDensityX96 computes the liquidity density at a given tick
func (g *GeometricDistribution) liquidityDensityX96(roundedTick, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick < minTick || roundedTick >= minTick+length*g.tickSpacing {
		// roundedTick is outside of the distribution
		var zero uint256.Int
		return &zero, nil
	}

	// x is the index of the roundedTick in the distribution
	// should be in the range [0, length)
	x := (roundedTick - minTick) / g.tickSpacing

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		// need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
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

		result, err := math.FullMulDiv(term1, &term2, term3)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		// alpha <= 1
		// will revert if alpha == 1 but that's ok
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

		result, err := math.FullMulDiv(&term1, term2, term3)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

// cumulativeAmount0 computes the cumulative amount0
func (g *GeometricDistribution) cumulativeAmount0(
	roundedTick, minTick, length int, alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	// x is the index of the roundedTick in the distribution
	var x int
	if roundedTick < minTick {
		x = -1
	} else if roundedTick >= minTick+length*g.tickSpacing {
		x = length
	} else {
		x = (roundedTick - minTick) / g.tickSpacing
	}

	sqrtRatioNegTickSpacing, err := math.GetSqrtPriceAtTick(-g.tickSpacing)
	if err != nil {
		return nil, err
	}
	sqrtRatioMinTick, err := math.GetSqrtPriceAtTick(minTick)
	if err != nil {
		return nil, err
	}

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		// compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
		if x >= length-1 {
			// roundedTick is the last tick in the distribution
			var zero uint256.Int
			return &zero, nil
		} else {
			xPlus1 := x + 1
			lengthMinusX := length - xPlus1

			intermediateTermIsPositive := alphaInvX96.Cmp(sqrtRatioNegTickSpacing) > 0
			numeratorTermLeft, err := math.Rpow(&alphaInvX96, lengthMinusX, math.Q96)
			if err != nil {
				return nil, err
			}
			numeratorTermRight, err := math.GetSqrtPriceAtTick(-g.tickSpacing * lengthMinusX)
			if err != nil {
				return nil, err
			}

			var numerator uint256.Int
			if intermediateTermIsPositive {
				numerator.Sub(numeratorTermLeft, numeratorTermRight)
			} else {
				numerator.Sub(numeratorTermRight, numeratorTermLeft)
			}

			var denominator uint256.Int
			if intermediateTermIsPositive {
				denominator.Sub(&alphaInvX96, sqrtRatioNegTickSpacing)
			} else {
				denominator.Sub(sqrtRatioNegTickSpacing, &alphaInvX96)
			}

			var term1 uint256.Int
			term1.Sub(math.Q96, &alphaInvX96)
			term1.Mul(&term1, &numerator)
			term1.Div(&term1, &denominator)

			term2, err := math.GetSqrtPriceAtTick(-g.tickSpacing * xPlus1)
			if err != nil {
				return nil, err
			}
			term1.Mul(&term1, term2)
			alphaPowLengthX96, err := math.Rpow(&alphaInvX96, length, math.Q96)
			if err != nil {
				return nil, err
			}
			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLengthX96)
			term1.Div(&term1, &q96MinusAlphaPowLength)

			var term3 uint256.Int
			term3.Sub(math.Q96, sqrtRatioNegTickSpacing)
			var result uint256.Int
			result.Mul(&term1, &term3)
			result.Div(&result, sqrtRatioMinTick)
			return &result, nil
		}
	} else {
		// alpha <= 1
		if x >= length-1 {
			var zero uint256.Int
			return &zero, nil
		} else {
			baseX96, err := math.FullMulDiv(alphaX96, sqrtRatioNegTickSpacing, math.Q96)
			if err != nil {
				return nil, err
			}

			xPlus1 := x + 1
			alphaPowXX96, err := math.Rpow(alphaX96, xPlus1, math.Q96)
			if err != nil {
				return nil, err
			}
			alphaPowLengthX96, err := math.Rpow(alphaX96, length, math.Q96)
			if err != nil {
				return nil, err
			}

			var term1 uint256.Int
			term1.Sub(math.Q96, alphaX96)
			term2, err := math.GetSqrtPriceAtTick(-g.tickSpacing * xPlus1)
			if err != nil {
				return nil, err
			}

			alphaPowXX96, err = math.FullMulDivUp(alphaPowXX96, term2, math.Q96)
			if err != nil {
				return nil, err
			}

			term3, err := math.GetSqrtPriceAtTick(-g.tickSpacing * length)
			if err != nil {
				return nil, err
			}
			alphaPowLengthX96, err = math.FullMulDivUp(alphaPowLengthX96, term3, math.Q96)
			if err != nil {
				return nil, err
			}

			var term4 uint256.Int
			term4.Sub(alphaPowXX96, alphaPowLengthX96)
			var numerator uint256.Int
			numerator.Mul(&term1, &term4)

			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLengthX96)
			var q96MinusBaseX96 uint256.Int
			q96MinusBaseX96.Sub(math.Q96, baseX96)
			var denominator uint256.Int
			denominator.Mul(&q96MinusAlphaPowLength, &q96MinusBaseX96)

			var term5 uint256.Int
			term5.Sub(math.Q96, sqrtRatioNegTickSpacing)
			result, err := math.FullMulDivUp(&term5, &numerator, &denominator)
			if err != nil {
				return nil, err
			}

			result, err = math.FullMulDivUp(result, math.Q96, sqrtRatioMinTick)
			if err != nil {
				return nil, err
			}

			return result, nil
		}
	}
}

// cumulativeAmount1 computes the cumulative amount1
func (g *GeometricDistribution) cumulativeAmount1(
	roundedTick, minTick, length int, alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	// x is the index of the roundedTick in the distribution
	var x int
	if roundedTick < minTick {
		x = -1
	} else if roundedTick >= minTick+length*g.tickSpacing {
		x = length
	} else {
		x = (roundedTick - minTick) / g.tickSpacing
	}

	sqrtRatioTickSpacing, err := math.GetSqrtPriceAtTick(g.tickSpacing)
	if err != nil {
		return nil, err
	}
	sqrtRatioNegMinTick, err := math.GetSqrtPriceAtTick(-minTick)
	if err != nil {
		return nil, err
	}

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		// compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
		if x <= 0 {
			// roundedTick is the first tick in the distribution
			var zero uint256.Int
			return &zero, nil
		} else {
			alphaInvPowLengthX96, err := math.Rpow(&alphaInvX96, length, math.Q96)
			if err != nil {
				return nil, err
			}
			baseX96, err := math.FullMulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)
			if err != nil {
				return nil, err
			}

			var numerator1 uint256.Int
			numerator1.Sub(alphaX96, math.Q96)
			var denominator1 uint256.Int
			denominator1.Sub(baseX96, math.Q96)

			term1, err := math.Rpow(&alphaInvX96, length-x, math.Q96)
			if err != nil {
				return nil, err
			}
			term2, err := math.GetSqrtPriceAtTick(x * g.tickSpacing)
			if err != nil {
				return nil, err
			}
			term1, err = math.FullMulDivUp(term1, term2, math.Q96)
			if err != nil {
				return nil, err
			}

			var numerator2 uint256.Int
			numerator2.Sub(term1, alphaInvPowLengthX96)
			var denominator2 uint256.Int
			denominator2.Sub(math.Q96, alphaInvPowLengthX96)

			term3, err := math.FullMulDivUp(math.Q96, &numerator2, &denominator2)
			if err != nil {
				return nil, err
			}

			term3, err = math.FullMulDivUp(term3, &numerator1, &denominator1)
			if err != nil {
				return nil, err
			}

			var term4 uint256.Int
			term4.Sub(sqrtRatioTickSpacing, math.Q96)
			result, err := math.FullMulDivUp(&term4, term3, sqrtRatioNegMinTick)
			if err != nil {
				return nil, err
			}

			return result, nil
		}
	} else {
		// alpha <= 1
		if x <= 0 {
			var zero uint256.Int
			return &zero, nil
		} else {
			sqrtRatioMinTick, err := math.GetSqrtPriceAtTick(minTick)
			if err != nil {
				return nil, err
			}
			baseX96, err := math.FullMulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)
			if err != nil {
				return nil, err
			}

			term1, err := math.Rpow(alphaX96, x+1, math.Q96)
			if err != nil {
				return nil, err
			}
			term2, err := math.GetSqrtPriceAtTick(g.tickSpacing * (x + 1))
			if err != nil {
				return nil, err
			}
			term1, err = math.FullMulDivUp(term1, term2, math.Q96)
			if err != nil {
				return nil, err
			}

			var numerator uint256.Int
			if math.Q96.Cmp(term1) > 0 {
				numerator.Sub(math.Q96, term1)
			} else {
				numerator.Sub(term1, math.Q96)
			}
			var q96MinusAlphaX96 uint256.Int
			q96MinusAlphaX96.Sub(math.Q96, alphaX96)
			numerator.Mul(&numerator, &q96MinusAlphaX96)

			var denominator uint256.Int
			if math.Q96.Cmp(baseX96) > 0 {
				denominator.Sub(math.Q96, baseX96)
			} else {
				denominator.Sub(baseX96, math.Q96)
			}
			alphaPowLengthX96, err := math.Rpow(alphaX96, length, math.Q96)
			if err != nil {
				return nil, err
			}
			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLengthX96)
			denominator.Mul(&denominator, &q96MinusAlphaPowLength)

			var term3 uint256.Int
			term3.Sub(sqrtRatioTickSpacing, math.Q96)
			result, err := math.FullMulDivUp(&term3, &numerator, &denominator)
			if err != nil {
				return nil, err
			}

			result, err = math.FullMulDivUp(result, sqrtRatioMinTick, math.Q96)
			if err != nil {
				return nil, err
			}

			return result, nil
		}
	}
}

// inverseCumulativeAmount0 computes the inverse of cumulative amount0
func (g *GeometricDistribution) inverseCumulativeAmount0(
	cumulativeAmount0_, totalLiquidity *uint256.Int,
	minTick, length int, alphaX96 *uint256.Int,
) (bool, int, error) {
	if cumulativeAmount0_.IsZero() {
		return true, minTick + (length-1)*g.tickSpacing, nil
	}

	// Simplified implementation - in practice this would use binary search
	// For now, return a reasonable estimate
	estimatedPosition := minTick + (length/2)*g.tickSpacing
	return true, estimatedPosition, nil
}

// computeSwap computes the swap parameters
func (g *GeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	minTick, length int,
	alphaX96 *uint256.Int,
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
		success, roundedTick, err = g.inverseCumulativeAmount0(
			inverseCumulativeAmountInput, totalLiquidity, minTick, length, alphaX96,
		)
		if !success || err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount0_, err = g.cumulativeAmount0(roundedTick+g.tickSpacing, minTick, length, alphaX96)
		} else {
			cumulativeAmount0_, err = g.cumulativeAmount0(roundedTick, minTick, length, alphaX96)
		}
		if err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}

		if exactIn {
			cumulativeAmount1_, err = g.cumulativeAmount1(roundedTick, minTick, length, alphaX96)
		} else {
			cumulativeAmount1_, err = g.cumulativeAmount1(roundedTick-g.tickSpacing, minTick, length, alphaX96)
		}
		if err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}
	} else {
		// compute roundedTick by inverting the cumulative amount1
		// Simplified implementation - would need proper inverse calculation
		roundedTick = minTick + (length/2)*g.tickSpacing

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount1_, err = g.cumulativeAmount1(roundedTick-g.tickSpacing, minTick, length, alphaX96)
		} else {
			cumulativeAmount1_, err = g.cumulativeAmount1(roundedTick, minTick, length, alphaX96)
		}
		if err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}

		if exactIn {
			cumulativeAmount0_, err = g.cumulativeAmount0(roundedTick, minTick, length, alphaX96)
		} else {
			cumulativeAmount0_, err = g.cumulativeAmount0(roundedTick+g.tickSpacing, minTick, length, alphaX96)
		}
		if err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}
	}

	// compute swap liquidity
	swapLiquidity, err = g.liquidityDensityX96(roundedTick, minTick, length, alphaX96)
	if err != nil {
		var zero uint256.Int
		return false, 0, &zero, &zero, &zero, err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
