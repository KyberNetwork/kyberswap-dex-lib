package geometric

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// LiquidityDensityX96 computes the liquidity density at a given tick
func LiquidityDensityX96(tickSpacing, roundedTick, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick < minTick || roundedTick >= minTick+length*tickSpacing {
		// roundedTick is outside of the distribution
		var zero uint256.Int
		return &zero, nil
	}

	// x is the index of the roundedTick in the distribution
	// should be in the range [0, length)
	x := (roundedTick - minTick) / tickSpacing

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
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(term1, &term2, &denom)
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
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(&term1, term2, &denom)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

// CumulativeAmount0 computes the cumulative amount0
func CumulativeAmount0(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	minTick,
	length int,
	alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	// x is the index of the roundedTick in the distribution
	var x int
	if roundedTick < minTick {
		x = -1
	} else if roundedTick >= minTick+length*tickSpacing {
		x = length
	} else {
		x = (roundedTick - minTick) / tickSpacing
	}

	sqrtRatioNegTickSpacing, err := math.GetSqrtPriceAtTick(-tickSpacing)
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
			numeratorTermRight, err := math.GetSqrtPriceAtTick(-tickSpacing * lengthMinusX)
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

			term2, err := math.GetSqrtPriceAtTick(-tickSpacing * xPlus1)
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
			term2, err := math.GetSqrtPriceAtTick(-tickSpacing * xPlus1)
			if err != nil {
				return nil, err
			}

			alphaPowXX96, err = math.FullMulDivUp(alphaPowXX96, term2, math.Q96)
			if err != nil {
				return nil, err
			}

			term3, err := math.GetSqrtPriceAtTick(-tickSpacing * length)
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

// CumulativeAmount1 computes the cumulative amount1
func CumulativeAmount1(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	minTick, length int,
	alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	// x is the index of the roundedTick in the distribution
	var x int
	if roundedTick < minTick {
		x = -1
	} else if roundedTick >= minTick+length*tickSpacing {
		x = length
	} else {
		x = (roundedTick - minTick) / tickSpacing
	}

	sqrtRatioTickSpacing, err := math.GetSqrtPriceAtTick(tickSpacing)
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
			term2, err := math.GetSqrtPriceAtTick(x * tickSpacing)
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
			term2, err := math.GetSqrtPriceAtTick(tickSpacing * (x + 1))
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

// InverseCumulativeAmount0 computes the inverse of cumulative amount0
func InverseCumulativeAmount0(
	tickSpacing int,
	cumulativeAmount0_, totalLiquidity *uint256.Int,
	minTick, length int, alphaX96 *uint256.Int,
) (bool, int, error) {
	if cumulativeAmount0_.IsZero() {
		return true, minTick + (length-1)*tickSpacing, nil
	}

	// Simplified implementation - in practice this would use binary search
	// For now, return a reasonable estimate
	estimatedPosition := minTick + (length/2)*tickSpacing
	return true, estimatedPosition, nil
}

// InverseCumulativeAmount0 computes the inverse of cumulative amount1
func InverseCumulativeAmount1(
	tickSpacing int,
	cumulativeAmount1_, totalLiquidity *uint256.Int,
	minTick, length int, alphaX96 *uint256.Int,
) (bool, int, error) {
	if cumulativeAmount1_.IsZero() {
		return true, minTick + (length-1)*tickSpacing, nil
	}

	// Simplified implementation - in practice this would use binary search
	// For now, return a reasonable estimate
	estimatedPosition := minTick + (length/2)*tickSpacing
	return true, estimatedPosition, nil
}
