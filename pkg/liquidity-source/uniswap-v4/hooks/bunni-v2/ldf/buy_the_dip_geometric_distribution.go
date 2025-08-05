package ldf

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// BuyTheDipGeometricDistribution represents a buy the dip geometric distribution LDF
type BuyTheDipGeometricDistribution struct {
	tickSpacing int
}

// NewBuyTheDipGeometricDistribution creates a new BuyTheDipGeometricDistribution
func NewBuyTheDipGeometricDistribution(tickSpacing int) ILiquidityDensityFunction {
	return &BuyTheDipGeometricDistribution{
		tickSpacing: tickSpacing,
	}
}

// decodeParams decodes the LDF parameters from bytes32
func (b *BuyTheDipGeometricDistribution) decodeParams(ldfParams [32]byte) (
	minTick, length, altThreshold int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThresholdDirection bool,
) {
	// | shiftMode - 1 byte | minTick - 3 bytes | length - 2 bytes | alpha - 4 bytes | altAlpha - 4 bytes | altThreshold - 3 bytes | altThresholdDirection - 1 byte |
	// minTick = int24(uint24(bytes3(ldfParams << 8)))
	minTick = int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))

	// length = int24(int16(uint16(bytes2(ldfParams << 32))))
	length = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))

	// uint256 alpha = uint32(bytes4(ldfParams << 48))
	alpha := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])
	// alphaX96 = alpha.mulDiv(Q96, ALPHA_BASE)
	alphaX96 = uint256.NewInt(uint64(alpha))
	alphaX96.Mul(alphaX96, math.Q96)
	alphaX96.Div(alphaX96, math.ALPHA_BASE)

	// uint256 altAlpha = uint32(bytes4(ldfParams << 80))
	altAlpha := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])
	// altAlphaX96 = altAlpha.mulDiv(Q96, ALPHA_BASE)
	altAlphaX96 = uint256.NewInt(uint64(altAlpha))
	altAlphaX96.Mul(altAlphaX96, math.Q96)
	altAlphaX96.Div(altAlphaX96, math.ALPHA_BASE)

	// altThreshold = int24(uint24(bytes3(ldfParams << 112)))
	altThreshold = int(int32(uint32(ldfParams[14])<<16 | uint32(ldfParams[15])<<8 | uint32(ldfParams[16])))

	// altThresholdDirection = uint8(bytes1(ldfParams << 136)) != 0
	altThresholdDirection = ldfParams[17] != 0

	return
}

// encodeState encodes the state into bytes32
func (b *BuyTheDipGeometricDistribution) encodeState(twapTick int) [32]byte {
	var state [32]byte
	state[0] = 1 // initialized = true
	state[1] = byte((twapTick >> 16) & 0xFF)
	state[2] = byte((twapTick >> 8) & 0xFF)
	state[3] = byte(twapTick & 0xFF)
	return state
}

// decodeBuyTheDipState decodes the LDF state from bytes32 for BuyTheDipGeometricDistribution
func decodeBuyTheDipState(ldfState [32]byte) (initialized bool, lastTwapTick int32) {
	// | initialized - 1 byte | lastTwapTick - 3 bytes |
	initialized = ldfState[0] == 1
	lastTwapTick = int32(uint32(ldfState[1])<<16 | uint32(ldfState[2])<<8 | uint32(ldfState[3]))
	return
}

// shouldUseAltAlpha determines if the alternative alpha should be used
func (b *BuyTheDipGeometricDistribution) shouldUseAltAlpha(twapTick, altThreshold int, altThresholdDirection bool) bool {
	if altThresholdDirection {
		return twapTick <= altThreshold
	}
	return twapTick >= altThreshold
}

// Query implements the Query method for BuyTheDipGeometricDistribution
func (b *BuyTheDipGeometricDistribution) Query(
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
	minTick, length, altThreshold, alphaX96, altAlphaX96, altThresholdDirection := b.decodeParams(ldfParams)
	initialized, lastTwapTick := decodeBuyTheDipState(ldfState)

	if initialized {
		// should surge if switched from one alpha to another
		shouldSurge = b.shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) !=
			b.shouldUseAltAlpha(int(lastTwapTick), altThreshold, altThresholdDirection)
	}

	// Determine which alpha to use based on TWAP threshold
	useAltAlpha := b.shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)
	currentAlpha := alphaX96
	if useAltAlpha {
		currentAlpha = altAlphaX96
	}

	// compute liquidityDensityX96
	liquidityDensityX96, err = b.liquidityDensityX96(roundedTick, minTick, length, currentAlpha)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	// compute cumulativeAmount0DensityX96
	cumulativeAmount0DensityX96, err = b.cumulativeAmount0(roundedTick+b.tickSpacing, minTick, length, currentAlpha)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	// compute cumulativeAmount1DensityX96
	cumulativeAmount1DensityX96, err = b.cumulativeAmount1(roundedTick-b.tickSpacing, minTick, length, currentAlpha)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = b.encodeState(twapTick)
	return
}

// liquidityDensityX96 computes the liquidity density at a given tick
func (b *BuyTheDipGeometricDistribution) liquidityDensityX96(
	roundedTick,
	minTick,
	length int,
	alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	if roundedTick < minTick || roundedTick >= minTick+length*b.tickSpacing {
		// roundedTick is outside of the distribution
		return uint256.NewInt(0), nil
	}

	// x is the index of the roundedTick in the distribution
	x := (roundedTick - minTick) / b.tickSpacing

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		// need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		// alphaInvX96.rpow(length-x, Q96)
		term1, err := math.Rpow(&alphaInvX96, length-x, math.Q96)
		if err != nil {
			return nil, err
		}

		// alphaX96 - Q96
		var term2 uint256.Int
		term2.Sub(alphaX96, math.Q96)

		// alphaInvX96.rpow(length, Q96)
		term3, err := math.Rpow(&alphaInvX96, length, math.Q96)
		if err != nil {
			return nil, err
		}

		// Q96 - alphaInvX96.rpow(length, Q96)
		var denominator uint256.Int
		denominator.Sub(math.Q96, term3)

		// term1 * term2 / denominator
		result, err := math.FullMulDiv(term1, &term2, &denominator)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		// alpha <= 1
		// will revert if alpha == 1 but that's ok
		var term1 uint256.Int
		term1.Sub(math.Q96, alphaX96)

		// alphaX96.rpow(x, Q96)
		term2, err := math.Rpow(alphaX96, x, math.Q96)
		if err != nil {
			return nil, err
		}

		// alphaX96.rpow(length, Q96)
		term3, err := math.Rpow(alphaX96, length, math.Q96)
		if err != nil {
			return nil, err
		}

		// Q96 - alphaX96.rpow(length, Q96)
		var denominator uint256.Int
		denominator.Sub(math.Q96, term3)

		// term1 * term2 / denominator
		result, err := math.FullMulDiv(&term1, term2, &denominator)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

// cumulativeAmount0 computes the cumulative amount0
func (b *BuyTheDipGeometricDistribution) cumulativeAmount0(
	roundedTick,
	minTick,
	length int,
	alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	// x is the index of the roundedTick in the distribution
	var x int
	if roundedTick < minTick {
		x = -1
	} else if roundedTick >= minTick+length*b.tickSpacing {
		x = length
	} else {
		x = (roundedTick - minTick) / b.tickSpacing
	}

	sqrtRatioNegTickSpacing, err := math.GetSqrtPriceAtTick(-b.tickSpacing)
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
			return uint256.NewInt(0), nil
		} else {
			xPlus1 := x + 1
			lengthMinusX := length - xPlus1

			intermediateTermIsPositive := alphaInvX96.Cmp(sqrtRatioNegTickSpacing) > 0
			numeratorTermLeft, err := math.Rpow(&alphaInvX96, lengthMinusX, math.Q96)
			if err != nil {
				return nil, err
			}
			numeratorTermRight, err := math.GetSqrtPriceAtTick(-b.tickSpacing * lengthMinusX)
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

			// (Q96 - alphaInvX96) * numerator / denominator
			var term1 uint256.Int
			term1.Sub(math.Q96, &alphaInvX96)
			term1.Mul(&term1, &numerator)
			term1.Div(&term1, &denominator)

			// term1 * getSqrtPriceAtTick(-tickSpacing * xPlus1)
			term2, err := math.GetSqrtPriceAtTick(-b.tickSpacing * xPlus1)
			if err != nil {
				return nil, err
			}
			term1.Mul(&term1, term2)

			// term1 / (Q96 - alphaInvX96.rpow(length, Q96))
			alphaPowLengthX96, err := math.Rpow(&alphaInvX96, length, math.Q96)
			if err != nil {
				return nil, err
			}
			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLengthX96)
			term1.Div(&term1, &q96MinusAlphaPowLength)

			// term1 * (Q96 - sqrtRatioNegTickSpacing) / sqrtRatioMinTick
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
			return uint256.NewInt(0), nil
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

			// (Q96 - alphaX96) * (alphaPowXX96 * getSqrtPriceAtTick(-tickSpacing * xPlus1) - alphaPowLengthX96 * getSqrtPriceAtTick(-tickSpacing * length))
			var term1 uint256.Int
			term1.Sub(math.Q96, alphaX96)

			term2, err := math.GetSqrtPriceAtTick(-b.tickSpacing * xPlus1)
			if err != nil {
				return nil, err
			}
			alphaPowXX96, err = math.FullMulDivUp(alphaPowXX96, term2, math.Q96)
			if err != nil {
				return nil, err
			}

			term3, err := math.GetSqrtPriceAtTick(-b.tickSpacing * length)
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

			// (Q96 - alphaPowLengthX96) * (Q96 - baseX96)
			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLengthX96)
			var q96MinusBaseX96 uint256.Int
			q96MinusBaseX96.Sub(math.Q96, baseX96)
			var denominator uint256.Int
			denominator.Mul(&q96MinusAlphaPowLength, &q96MinusBaseX96)

			// (Q96 - sqrtRatioNegTickSpacing) * numerator / denominator
			var term5 uint256.Int
			term5.Sub(math.Q96, sqrtRatioNegTickSpacing)
			result, err := math.FullMulDivUp(&term5, &numerator, &denominator)
			if err != nil {
				return nil, err
			}

			// result * Q96 / sqrtRatioMinTick
			result, err = math.FullMulDivUp(result, math.Q96, sqrtRatioMinTick)
			if err != nil {
				return nil, err
			}

			return result, nil
		}
	}
}

// cumulativeAmount1 computes the cumulative amount1
func (b *BuyTheDipGeometricDistribution) cumulativeAmount1(
	roundedTick,
	minTick,
	length int,
	alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	// x is the index of the roundedTick in the distribution
	var x int
	if roundedTick < minTick {
		x = -1
	} else if roundedTick >= minTick+length*b.tickSpacing {
		x = length
	} else {
		x = (roundedTick - minTick) / b.tickSpacing
	}

	sqrtRatioTickSpacing, err := math.GetSqrtPriceAtTick(b.tickSpacing)
	if err != nil {
		return nil, err
	}
	sqrtRatioNegMinTick, err := math.GetSqrtPriceAtTick(-minTick)
	if err != nil {
		return nil, err
	}

	if alphaX96.Gt(math.Q96) {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		// compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
		if x <= 0 {
			// roundedTick is the first tick in the distribution
			return uint256.NewInt(0), nil
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

			// alphaInvX96.rpow(length-x, Q96) * getSqrtPriceAtTick(x * tickSpacing) / Q96
			term1, err := math.Rpow(&alphaInvX96, length-x, math.Q96)
			if err != nil {
				return nil, err
			}
			term2, err := math.GetSqrtPriceAtTick(x * b.tickSpacing)
			if err != nil {
				return nil, err
			}
			term1, err = math.FullMulDivUp(term1, term2, math.Q96)
			if err != nil {
				return nil, err
			}

			// (term1 - alphaInvPowLengthX96) / (Q96 - alphaInvPowLengthX96)
			var numerator2 uint256.Int
			numerator2.Sub(term1, alphaInvPowLengthX96)
			var denominator2 uint256.Int
			denominator2.Sub(math.Q96, alphaInvPowLengthX96)

			// Q96 * numerator2 / denominator2
			term3, err := math.FullMulDivUp(math.Q96, &numerator2, &denominator2)
			if err != nil {
				return nil, err
			}

			// term3 * numerator1 / denominator1
			term3, err = math.FullMulDivUp(term3, &numerator1, &denominator1)
			if err != nil {
				return nil, err
			}

			// term3 * (sqrtRatioTickSpacing - Q96) / sqrtRatioNegMinTick
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
			return uint256.NewInt(0), nil
		} else {
			sqrtRatioMinTick, err := math.GetSqrtPriceAtTick(minTick)
			if err != nil {
				return nil, err
			}
			baseX96, err := math.FullMulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)
			if err != nil {
				return nil, err
			}

			// alphaX96.rpow(x+1, Q96) * getSqrtPriceAtTick(tickSpacing * (x + 1)) / Q96
			term1, err := math.Rpow(alphaX96, x+1, math.Q96)
			if err != nil {
				return nil, err
			}
			term2, err := math.GetSqrtPriceAtTick(b.tickSpacing * (x + 1))
			if err != nil {
				return nil, err
			}
			term1, err = math.FullMulDivUp(term1, term2, math.Q96)
			if err != nil {
				return nil, err
			}

			// dist(Q96, term1) * (Q96 - alphaX96)
			var numerator uint256.Int
			if math.Q96.Gt(term1) {
				numerator.Sub(math.Q96, term1)
			} else {
				numerator.Sub(term1, math.Q96)
			}
			var q96MinusAlphaX96 uint256.Int
			q96MinusAlphaX96.Sub(math.Q96, alphaX96)
			numerator.Mul(&numerator, &q96MinusAlphaX96)

			// dist(Q96, baseX96) * (Q96 - alphaX96.rpow(length, Q96))
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

			// (sqrtRatioTickSpacing - Q96) * numerator / denominator
			var term3 uint256.Int
			term3.Sub(sqrtRatioTickSpacing, math.Q96)
			result, err := math.FullMulDivUp(&term3, &numerator, &denominator)
			if err != nil {
				return nil, err
			}

			// result * sqrtRatioMinTick / Q96
			result, err = math.FullMulDivUp(result, sqrtRatioMinTick, math.Q96)
			if err != nil {
				return nil, err
			}

			return result, nil
		}
	}
}

// inverseCumulativeAmount0 computes the inverse of cumulativeAmount0
func (b *BuyTheDipGeometricDistribution) inverseCumulativeAmount0(
	cumulativeAmount0_,
	_ *uint256.Int,
	twapTick, minTick, length int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (bool, int, error) {
	// Determine which alpha to use based on TWAP threshold
	useAltAlpha := b.shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)
	currentAlpha := alphaX96
	if useAltAlpha {
		currentAlpha = altAlphaX96
	}

	// Binary search to find the rounded tick
	left := minTick
	right := minTick + length*b.tickSpacing

	for left <= right {
		mid := left + (right-left)/2
		// Round to nearest tick spacing
		mid = (mid / b.tickSpacing) * b.tickSpacing

		cumAmount, err := b.cumulativeAmount0(mid, minTick, length, currentAlpha)
		if err != nil {
			return false, 0, err
		}

		if cumAmount.Cmp(cumulativeAmount0_) >= 0 {
			right = mid - b.tickSpacing
		} else {
			left = mid + b.tickSpacing
		}
	}

	// Return the largest rounded tick whose cumulativeAmount0 is >= input
	return true, right, nil
}

// inverseCumulativeAmount1 computes the inverse of cumulativeAmount1
func (b *BuyTheDipGeometricDistribution) inverseCumulativeAmount1(
	cumulativeAmount1_,
	totalLiquidity *uint256.Int,
	twapTick, minTick, length int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (bool, int, error) {
	// Determine which alpha to use based on TWAP threshold
	useAltAlpha := b.shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)
	currentAlpha := alphaX96
	if useAltAlpha {
		currentAlpha = altAlphaX96
	}

	// Binary search to find the rounded tick
	left := minTick
	right := minTick + length*b.tickSpacing

	for left <= right {
		mid := left + (right-left)/2
		// Round to nearest tick spacing
		mid = (mid / b.tickSpacing) * b.tickSpacing

		cumAmount, err := b.cumulativeAmount1(mid, minTick, length, currentAlpha)
		if err != nil {
			return false, 0, err
		}

		if cumAmount.Cmp(cumulativeAmount1_) >= 0 {
			left = mid + b.tickSpacing
		} else {
			right = mid - b.tickSpacing
		}
	}

	// Return the smallest rounded tick whose cumulativeAmount1 is >= input
	return true, left, nil
}

func (b *BuyTheDipGeometricDistribution) ComputeSwap(
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
	minTick, length, altThreshold, alphaX96, altAlphaX96, altThresholdDirection := b.decodeParams(ldfParams)

	return b.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
}

// computeSwap computes the swap parameters
func (b *BuyTheDipGeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	twapTick, minTick, length int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	// Determine which alpha to use based on TWAP threshold
	useAltAlpha := b.shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)
	currentAlpha := alphaX96
	if useAltAlpha {
		currentAlpha = altAlphaX96
	}

	if exactIn == zeroForOne {
		// compute roundedTick by inverting the cumulative amount0
		success, roundedTick, err = b.inverseCumulativeAmount0(
			inverseCumulativeAmountInput,
			totalLiquidity,
			twapTick, minTick, length,
			alphaX96, altAlphaX96,
			altThreshold, altThresholdDirection,
		)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount0_, err = b.cumulativeAmount0(roundedTick+b.tickSpacing, minTick, length, currentAlpha)
		} else {
			cumulativeAmount0_, err = b.cumulativeAmount0(roundedTick, minTick, length, currentAlpha)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount1_, err = b.cumulativeAmount1(roundedTick, minTick, length, currentAlpha)
		} else {
			cumulativeAmount1_, err = b.cumulativeAmount1(roundedTick-b.tickSpacing, minTick, length, currentAlpha)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	} else {
		// compute roundedTick by inverting the cumulative amount1
		success, roundedTick, err = b.inverseCumulativeAmount1(
			inverseCumulativeAmountInput,
			totalLiquidity,
			twapTick, minTick, length,
			alphaX96, altAlphaX96,
			altThreshold, altThresholdDirection,
		)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount1_, err = b.cumulativeAmount1(roundedTick-b.tickSpacing, minTick, length, currentAlpha)
		} else {
			cumulativeAmount1_, err = b.cumulativeAmount1(roundedTick, minTick, length, currentAlpha)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount0_, err = b.cumulativeAmount0(roundedTick, minTick, length, currentAlpha)
		} else {
			cumulativeAmount0_, err = b.cumulativeAmount0(roundedTick+b.tickSpacing, minTick, length, currentAlpha)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	}

	// compute swap liquidity
	swapLiquidity, err = b.liquidityDensityX96(roundedTick, minTick, length, currentAlpha)
	if err != nil {
		return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
