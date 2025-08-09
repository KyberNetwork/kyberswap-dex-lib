package ldf

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/shift-mode"
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
		minTick = shiftmode.EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
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
		minTick = shiftmode.EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
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
func (g *GeometricDistribution) decodeParams(
	twapTick int,
	ldfParams [32]byte,
) (
	minTick,
	length int,
	alphaX96 *uint256.Int,
	shiftMode shiftmode.ShiftMode,
) {
	// | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length - 2 bytes | alpha - 4 bytes |
	shiftMode = shiftmode.ShiftMode(ldfParams[0])
	length = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))
	alpha := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])

	// Convert alpha to alphaX96 (alpha * Q96 / ALPHA_BASE)
	alphaX96 = uint256.NewInt(uint64(alpha))
	alphaX96.Mul(alphaX96, math.Q96)
	alphaX96.Div(alphaX96, math.ALPHA_BASE)

	if shiftMode != shiftmode.Static {
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
	roundedTick,
	minTick,
	length int,
	alphaX96 *uint256.Int,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	// liquidityDensityX96
	if roundedTick < minTick || roundedTick >= minTick+length*g.tickSpacing {
		var zero uint256.Int
		liquidityDensityX96 = &zero
	} else {
		x := (roundedTick - minTick) / g.tickSpacing
		if alphaX96.Gt(math.Q96) {
			var alphaInvX96 uint256.Int
			alphaInvX96.Mul(math.Q96, math.Q96)
			alphaInvX96.Div(&alphaInvX96, alphaX96)

			term1, err1 := math.Rpow(&alphaInvX96, length-x, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			var term2 uint256.Int
			term2.Sub(alphaX96, math.Q96)
			term3, err1 := math.Rpow(&alphaInvX96, length, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			var denom uint256.Int
			denom.Sub(math.Q96, term3)

			liquidityDensityX96, err = math.FullMulDiv(term1, &term2, &denom)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			var term1 uint256.Int
			term1.Sub(math.Q96, alphaX96)
			term2, err1 := math.Rpow(alphaX96, x, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			term3, err1 := math.Rpow(alphaX96, length, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			var denom uint256.Int
			denom.Sub(math.Q96, term3)

			liquidityDensityX96, err = math.FullMulDiv(&term1, term2, &denom)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	// x for cumulative amounts
	var x int
	if roundedTick < minTick {
		x = -1
	} else if roundedTick >= minTick+length*g.tickSpacing {
		x = length
	} else {
		x = (roundedTick - minTick) / g.tickSpacing
	}

	// precompute sqrt ratios
	sqrtRatioTickSpacing, err := math.GetSqrtPriceAtTick(g.tickSpacing)
	if err != nil {
		return nil, nil, nil, err
	}
	sqrtRatioNegTickSpacing, err := math.GetSqrtPriceAtTick(-g.tickSpacing)
	if err != nil {
		return nil, nil, nil, err
	}
	sqrtRatioMinTick, err := math.GetSqrtPriceAtTick(minTick)
	if err != nil {
		return nil, nil, nil, err
	}
	sqrtRatioNegMinTick, err := math.GetSqrtPriceAtTick(-minTick)
	if err != nil {
		return nil, nil, nil, err
	}

	if alphaX96.Gt(math.Q96) {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		// cumulativeAmount0DensityX96 (to the right)
		if x >= length-1 {
			var zero uint256.Int
			cumulativeAmount0DensityX96 = &zero
		} else {
			xPlus1 := x + 1
			lengthMinusX := length - xPlus1

			intermediateTermIsPositive := alphaInvX96.Cmp(sqrtRatioNegTickSpacing) > 0
			numeratorTermLeft, err1 := math.Rpow(&alphaInvX96, lengthMinusX, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			numeratorTermRight, err1 := math.GetSqrtPriceAtTick(-g.tickSpacing * lengthMinusX)
			if err1 != nil {
				return nil, nil, nil, err1
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

			var q96MinusAlphaInv uint256.Int
			q96MinusAlphaInv.Sub(math.Q96, &alphaInvX96)
			term1, err1 := math.FullMulDivUp(&q96MinusAlphaInv, &numerator, &denominator)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			term2, err1 := math.GetSqrtPriceAtTick(-g.tickSpacing * xPlus1)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			alphaInvPowLengthX96, err1 := math.Rpow(&alphaInvX96, length, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			var q96MinusAlphaInvPowLen uint256.Int
			q96MinusAlphaInvPowLen.Sub(math.Q96, alphaInvPowLengthX96)
			term1, err1 = math.FullMulDivUp(term1, term2, &q96MinusAlphaInvPowLen)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			var q96MinusSqrtNegSpacing uint256.Int
			q96MinusSqrtNegSpacing.Sub(math.Q96, sqrtRatioNegTickSpacing)
			cumulativeAmount0DensityX96, err = math.FullMulDivUp(term1, &q96MinusSqrtNegSpacing, sqrtRatioMinTick)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		// cumulativeAmount1DensityX96 (to the left)
		if x <= 0 {
			var zero uint256.Int
			cumulativeAmount1DensityX96 = &zero
		} else {
			alphaInvPowLengthX96, err1 := math.Rpow(&alphaInvX96, length, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			baseX96, err1 := math.FullMulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			var numerator1 uint256.Int
			numerator1.Sub(alphaX96, math.Q96)
			var denominator1 uint256.Int
			denominator1.Sub(baseX96, math.Q96)

			term1, err1 := math.Rpow(&alphaInvX96, length-x, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			term2, err1 := math.GetSqrtPriceAtTick(x * g.tickSpacing)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			term1, err1 = math.FullMulDivUp(term1, term2, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			var numerator2 uint256.Int
			numerator2.Sub(term1, alphaInvPowLengthX96)
			var denominator2 uint256.Int
			denominator2.Sub(math.Q96, alphaInvPowLengthX96)

			term3, err1 := math.FullMulDivUp(math.Q96, &numerator2, &denominator2)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			term3, err1 = math.FullMulDivUp(term3, &numerator1, &denominator1)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			var sqrtTickMinusQ96 uint256.Int
			sqrtTickMinusQ96.Sub(sqrtRatioTickSpacing, math.Q96)
			cumulativeAmount1DensityX96, err = math.FullMulDivUp(&sqrtTickMinusQ96, term3, sqrtRatioNegMinTick)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	} else {
		// alpha <= 1
		// cumulativeAmount0DensityX96 (to the right)
		if x >= length-1 {
			var zero uint256.Int
			cumulativeAmount0DensityX96 = &zero
		} else {
			baseX96, err1 := math.FullMulDiv(alphaX96, sqrtRatioNegTickSpacing, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			xPlus1 := x + 1
			alphaPowXX96, err1 := math.Rpow(alphaX96, xPlus1, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			alphaPowLengthX96, err1 := math.Rpow(alphaX96, length, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			var q96MinusAlpha uint256.Int
			q96MinusAlpha.Sub(math.Q96, alphaX96)
			term2, err1 := math.GetSqrtPriceAtTick(-g.tickSpacing * xPlus1)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			alphaPowXX96, err1 = math.FullMulDivUp(alphaPowXX96, term2, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			term3, err1 := math.GetSqrtPriceAtTick(-g.tickSpacing * length)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			alphaPowLengthX96, err1 = math.FullMulDivUp(alphaPowLengthX96, term3, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			var diff uint256.Int
			diff.Sub(alphaPowXX96, alphaPowLengthX96)
			var numerator uint256.Int
			numerator.Mul(&q96MinusAlpha, &diff)

			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLengthX96)
			var q96MinusBaseX96 uint256.Int
			q96MinusBaseX96.Sub(math.Q96, baseX96)
			var denominator uint256.Int
			denominator.Mul(&q96MinusAlphaPowLength, &q96MinusBaseX96)

			var q96MinusSqrtNegSpacing uint256.Int
			q96MinusSqrtNegSpacing.Sub(math.Q96, sqrtRatioNegTickSpacing)
			result, err1 := math.FullMulDivUp(&q96MinusSqrtNegSpacing, &numerator, &denominator)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			cumulativeAmount0DensityX96, err = math.FullMulDivUp(result, math.Q96, sqrtRatioMinTick)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		// cumulativeAmount1DensityX96 (to the left)
		if x <= 0 {
			var zero uint256.Int
			cumulativeAmount1DensityX96 = &zero
		} else {
			sqrtRatioMinTick2, err1 := math.GetSqrtPriceAtTick(minTick)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			baseX96, err1 := math.FullMulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			term1, err1 := math.Rpow(alphaX96, x+1, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			term2, err1 := math.GetSqrtPriceAtTick(g.tickSpacing * (x + 1))
			if err1 != nil {
				return nil, nil, nil, err1
			}
			term1, err1 = math.FullMulDivUp(term1, term2, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
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
			alphaPowLengthX96, err1 := math.Rpow(alphaX96, length, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLengthX96)
			denominator.Mul(&denominator, &q96MinusAlphaPowLength)

			var sqrtTickMinusQ96 uint256.Int
			sqrtTickMinusQ96.Sub(sqrtRatioTickSpacing, math.Q96)
			result, err1 := math.FullMulDivUp(&sqrtTickMinusQ96, &numerator, &denominator)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			cumulativeAmount1DensityX96, err = math.FullMulDivUp(result, sqrtRatioMinTick2, math.Q96)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	return
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
		success, roundedTick, err = geoLib.InverseCumulativeAmount0(
			g.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			minTick,
			length,
			alphaX96,
		)
		if !success || err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount0_, err = geoLib.CumulativeAmount0(
				g.tickSpacing,
				roundedTick+g.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
		} else {
			cumulativeAmount0_, err = geoLib.CumulativeAmount0(
				g.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
		}
		if err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}

		if exactIn {
			cumulativeAmount1_, err = geoLib.CumulativeAmount1(
				g.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
		} else {
			cumulativeAmount1_, err = geoLib.CumulativeAmount1(
				g.tickSpacing,
				roundedTick-g.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
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
			cumulativeAmount1_, err = geoLib.CumulativeAmount1(
				g.tickSpacing,
				roundedTick-g.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
		} else {
			cumulativeAmount1_, err = geoLib.CumulativeAmount1(
				g.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
		}
		if err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}

		if exactIn {
			cumulativeAmount0_, err = geoLib.CumulativeAmount0(
				g.tickSpacing,
				roundedTick,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
		} else {
			cumulativeAmount0_, err = geoLib.CumulativeAmount0(
				g.tickSpacing,
				roundedTick+g.tickSpacing,
				totalLiquidity,
				minTick,
				length,
				alphaX96,
			)
		}
		if err != nil {
			var zero uint256.Int
			return false, 0, &zero, &zero, &zero, err
		}
	}

	// compute swap liquidity
	swapLiquidity, err = geoLib.LiquidityDensityX96(g.tickSpacing, roundedTick, minTick, length, alphaX96)
	if err != nil {
		var zero uint256.Int
		return false, 0, &zero, &zero, &zero, err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
