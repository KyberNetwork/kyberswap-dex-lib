package ldf

import (
	geoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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

	newLdfState = EncodeState(minTick)
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

// query computes the liquidity density and cumulative amounts based on Solidity logic
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
	// compute liquidityDensityX96 using lib function
	liquidityDensityX96, err = geoLib.LiquidityDensityX96(g.tickSpacing, roundedTick, minTick, length, alphaX96)
	if err != nil {
		return nil, nil, nil, err
	}

	// x is the index of the roundedTick in the distribution
	// should be in the range [0, length)
	var x int
	if roundedTick < minTick {
		// roundedTick is to the left of the distribution
		// set x = -1
		x = -1
	} else if roundedTick >= minTick+length*g.tickSpacing {
		// roundedTick is to the right of the distribution
		// set x = length
		x = length
	} else {
		// roundedTick is in the distribution
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

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		// compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
		if x >= length-1 {
			// roundedTick is the last tick in the distribution
			// cumulativeAmount0DensityX96 is just 0
			cumulativeAmount0DensityX96 = uint256.NewInt(0)
		} else {
			xPlus1 := x + 1 // the rounded tick to the right of the current rounded tick

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

		// compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
		if x <= 0 {
			// roundedTick is the first tick in the distribution
			// cumulativeAmount1DensityX96 is just 0
			cumulativeAmount1DensityX96 = uint256.NewInt(0)
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
		// compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
		if x >= length-1 {
			// roundedTick is the last tick in the distribution
			// cumulativeAmount0DensityX96 is just 0
			cumulativeAmount0DensityX96 = uint256.NewInt(0)
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

		// compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
		if x <= 0 {
			// roundedTick is the first tick in the distribution
			// cumulativeAmount1DensityX96 is just 0
			cumulativeAmount1DensityX96 = uint256.NewInt(0)
		} else {
			baseX96, err1 := math.FullMulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}

			term1, err1 := math.Rpow(alphaX96, x, math.Q96)
			if err1 != nil {
				return nil, nil, nil, err1
			}
			term2, err1 := math.GetSqrtPriceAtTick(g.tickSpacing * x)
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
			cumulativeAmount1DensityX96, err = math.FullMulDivUp(result, sqrtRatioMinTick, math.Q96)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	return liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, nil
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
			return false, 0, u256.U0, u256.U0, u256.U0, err
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
			return false, 0, u256.U0, u256.U0, u256.U0, err
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
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}
	} else {
		// compute roundedTick by inverting the cumulative amount1
		success, roundedTick, err = geoLib.InverseCumulativeAmount1(
			g.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			minTick,
			length,
			alphaX96,
		)
		if !success || err != nil {
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}

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
			return false, 0, u256.U0, u256.U0, u256.U0, err
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
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}
	}

	// compute swap liquidity
	swapLiquidity, err = geoLib.LiquidityDensityX96(g.tickSpacing, roundedTick, minTick, length, alphaX96)
	if err != nil {
		return false, 0, u256.U0, u256.U0, u256.U0, err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
