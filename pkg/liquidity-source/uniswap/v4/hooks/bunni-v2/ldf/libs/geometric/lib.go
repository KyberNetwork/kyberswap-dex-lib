package geometric

import (
	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// DecodeParams decodes the LDF parameters from bytes32
func DecodeParams(
	tickSpacing,
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

	alphaX96 = math.MulDiv(uint256.NewInt(uint64(alpha)), math.Q96, math.ALPHA_BASE)

	if shiftMode != shiftmode.Static {
		offsetRaw := uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])

		if offsetRaw&0x800000 != 0 {
			offsetRaw |= 0xFF000000
		}
		offset := int(int32(offsetRaw))

		minTick = math.RoundTickSingle(twapTick+offset, tickSpacing)

		minUsableTick := math.MinUsableTick(tickSpacing)
		maxUsableTick := math.MaxUsableTick(tickSpacing)
		if minTick < minUsableTick {
			minTick = minUsableTick
		} else if minTick > maxUsableTick-length*tickSpacing {
			minTick = maxUsableTick - length*tickSpacing
		}
	} else {
		minTickRaw := uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])

		if minTickRaw&0x800000 != 0 {
			minTickRaw |= 0xFF000000
		}
		minTick = int(int32(minTickRaw))
	}

	return
}

// LiquidityDensityX96 computes the liquidity density at a given tick
func LiquidityDensityX96(tickSpacing, roundedTick, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick < minTick || roundedTick >= minTick+length*tickSpacing {
		return uint256.NewInt(0), nil
	}

	x := (roundedTick - minTick) / tickSpacing

	if alphaX96.Gt(math.Q96) {
		alphaInvX96 := math.MulDiv(math.Q96, math.Q96, alphaX96)

		term1, err := math.Rpow(alphaInvX96, length-x, math.Q96)
		if err != nil {
			return nil, err
		}

		var alphaMinusQ96 uint256.Int
		alphaMinusQ96.Sub(alphaX96, math.Q96)

		alphaInvPowLength, err := math.Rpow(alphaInvX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var q96MinusAlphaInvPowLength uint256.Int
		q96MinusAlphaInvPowLength.Sub(math.Q96, alphaInvPowLength)

		return math.FullMulDiv(term1, &alphaMinusQ96, &q96MinusAlphaInvPowLength)
	} else {
		var q96MinusAlpha uint256.Int
		q96MinusAlpha.Sub(math.Q96, alphaX96)

		alphaPowX, err := math.Rpow(alphaX96, x, math.Q96)
		if err != nil {
			return nil, err
		}

		alphaPowLength, err := math.Rpow(alphaX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var q96MinusAlphaPowLength uint256.Int
		q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLength)

		return math.MulDiv(&q96MinusAlpha, alphaPowX, &q96MinusAlphaPowLength), nil
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
	var x int
	if roundedTick < minTick {
		x = 0
	} else if roundedTick >= minTick+length*tickSpacing {
		return uint256.NewInt(0), nil
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

	var cumulativeAmount0DensityX96 *uint256.Int

	if alphaX96.Gt(math.Q96) {
		alphaInvX96 := math.MulDiv(math.Q96, math.Q96, alphaX96)

		if x >= length {
			cumulativeAmount0DensityX96 = uint256.NewInt(0)
		} else {
			lengthMinusX := length - x

			intermediateTermIsPositive := alphaInvX96.Gt(sqrtRatioNegTickSpacing)
			numeratorTermLeft, err := math.Rpow(alphaInvX96, lengthMinusX, math.Q96)
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
				denominator.Sub(alphaInvX96, sqrtRatioNegTickSpacing)
			} else {
				denominator.Sub(sqrtRatioNegTickSpacing, alphaInvX96)
			}

			var q96MinusAlphaInv uint256.Int
			q96MinusAlphaInv.Sub(math.Q96, alphaInvX96)
			term1 := math.MulDivUp(&q96MinusAlphaInv, &numerator, &denominator)

			sqrtPriceAtNegTickX, err := math.GetSqrtPriceAtTick(-tickSpacing * x)
			if err != nil {
				return nil, err
			}

			alphaPowLength, err := math.Rpow(alphaInvX96, length, math.Q96)
			if err != nil {
				return nil, err
			}
			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLength)

			term2 := math.MulDivUp(term1, sqrtPriceAtNegTickX, &q96MinusAlphaPowLength)

			var q96MinusSqrtRatioNegTick uint256.Int
			q96MinusSqrtRatioNegTick.Sub(math.Q96, sqrtRatioNegTickSpacing)
			cumulativeAmount0DensityX96 = math.MulDivUp(term2, &q96MinusSqrtRatioNegTick, sqrtRatioMinTick)
		}
	} else {
		if x >= length {
			cumulativeAmount0DensityX96 = uint256.NewInt(0)
		} else {
			baseX96 := math.MulDiv(alphaX96, sqrtRatioNegTickSpacing, math.Q96)

			alphaPowX, err := math.Rpow(alphaX96, x, math.Q96)
			if err != nil {
				return nil, err
			}
			alphaPowLength, err := math.Rpow(alphaX96, length, math.Q96)
			if err != nil {
				return nil, err
			}

			sqrtPriceAtNegTickX, err := math.GetSqrtPriceAtTick(-tickSpacing * x)
			if err != nil {
				return nil, err
			}
			alphaPowXTerm, err := math.FullMulDivUp(alphaPowX, sqrtPriceAtNegTickX, math.Q96)
			if err != nil {
				return nil, err
			}

			sqrtPriceAtNegTickLength, err := math.GetSqrtPriceAtTick(-tickSpacing * length)
			if err != nil {
				return nil, err
			}
			alphaPowLengthTerm, err := math.FullMulDivUp(alphaPowLength, sqrtPriceAtNegTickLength, math.Q96)
			if err != nil {
				return nil, err
			}

			var q96MinusAlpha uint256.Int
			q96MinusAlpha.Sub(math.Q96, alphaX96)

			var numerator uint256.Int
			numerator.Sub(alphaPowXTerm, alphaPowLengthTerm).Mul(&q96MinusAlpha, &numerator)

			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLength)

			var denominator uint256.Int
			denominator.Sub(math.Q96, baseX96).Mul(&q96MinusAlphaPowLength, &denominator)

			var q96MinusSqrtRatioNegTick uint256.Int
			q96MinusSqrtRatioNegTick.Sub(math.Q96, sqrtRatioNegTickSpacing)
			result, err := math.FullMulDivUp(&q96MinusSqrtRatioNegTick, &numerator, &denominator)
			if err != nil {
				return nil, err
			}

			cumulativeAmount0DensityX96, err = math.FullMulDivUp(result, math.Q96, sqrtRatioMinTick)
			if err != nil {
				return nil, err
			}
		}
	}

	return math.FullMulX96Up(cumulativeAmount0DensityX96, totalLiquidity)
}

// CumulativeAmount1 computes the cumulative amount1
func CumulativeAmount1(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	minTick, length int,
	alphaX96 *uint256.Int,
) (*uint256.Int, error) {
	var x int
	if roundedTick < minTick {
		return uint256.NewInt(0), nil
	} else if roundedTick >= minTick+length*tickSpacing {
		x = length - 1
	} else {
		x = (roundedTick - minTick) / tickSpacing
	}

	sqrtRatioTickSpacing, err := math.GetSqrtPriceAtTick(tickSpacing)
	if err != nil {
		return nil, err
	}

	var cumulativeAmount1DensityX96 *uint256.Int

	if alphaX96.Gt(math.Q96) {
		alphaInvX96 := math.MulDiv(math.Q96, math.Q96, alphaX96)

		if x < 0 {
			cumulativeAmount1DensityX96 = uint256.NewInt(0)
		} else {
			alphaInvPowLengthX96, err := math.Rpow(alphaInvX96, length, math.Q96)
			if err != nil {
				return nil, err
			}
			sqrtRatioNegMinTick, err := math.GetSqrtPriceAtTick(-minTick)
			if err != nil {
				return nil, err
			}

			baseX96 := math.MulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)

			var numerator1 uint256.Int
			numerator1.Sub(alphaX96, math.Q96)

			var denominator1 uint256.Int
			denominator1.Sub(baseX96, math.Q96)

			lengthMinusXMinus1 := length - x - 1
			alphaPowTerm, err := math.Rpow(alphaInvX96, lengthMinusXMinus1, math.Q96)
			if err != nil {
				return nil, err
			}

			sqrtPriceAtXPlus1Tick, err := math.GetSqrtPriceAtTick((x + 1) * tickSpacing)
			if err != nil {
				return nil, err
			}

			alphaPowTerm, err = math.FullMulDivUp(alphaPowTerm, sqrtPriceAtXPlus1Tick, math.Q96)
			if err != nil {
				return nil, err
			}

			var numerator2 uint256.Int
			numerator2.Sub(alphaPowTerm, alphaInvPowLengthX96)

			var denominator2 uint256.Int
			denominator2.Sub(math.Q96, alphaInvPowLengthX96)

			term1 := math.MulDivUp(math.Q96, &numerator2, &denominator2)
			term2 := math.MulDivUp(term1, &numerator1, &denominator1)

			var sqrtRatioTickSpacingMinusQ96 uint256.Int
			sqrtRatioTickSpacingMinusQ96.Sub(sqrtRatioTickSpacing, math.Q96)
			cumulativeAmount1DensityX96 = math.MulDivUp(term2, &sqrtRatioTickSpacingMinusQ96, sqrtRatioNegMinTick)
		}
	} else {
		if x < 0 {
			cumulativeAmount1DensityX96 = uint256.NewInt(0)
		} else {
			sqrtRatioMinTick, err := math.GetSqrtPriceAtTick(minTick)
			if err != nil {
				return nil, err
			}

			baseX96 := math.MulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)

			alphaPowXPlus1, err := math.Rpow(alphaX96, x+1, math.Q96)
			if err != nil {
				return nil, err
			}

			sqrtPriceAtTickSpacingXPlus1, err := math.GetSqrtPriceAtTick(tickSpacing * (x + 1))
			if err != nil {
				return nil, err
			}

			alphaPowTerm, err := math.FullMulDivUp(alphaPowXPlus1, sqrtPriceAtTickSpacingXPlus1, math.Q96)
			if err != nil {
				return nil, err
			}

			distQ96AlphaPow := math.Dist(math.Q96, alphaPowTerm)
			var q96MinusAlpha uint256.Int
			q96MinusAlpha.Sub(math.Q96, alphaX96)
			var numerator uint256.Int
			numerator.Mul(distQ96AlphaPow, &q96MinusAlpha)

			distQ96Base := math.Dist(math.Q96, baseX96)
			alphaPowLength, err := math.Rpow(alphaX96, length, math.Q96)
			if err != nil {
				return nil, err
			}
			var q96MinusAlphaPowLength uint256.Int
			q96MinusAlphaPowLength.Sub(math.Q96, alphaPowLength)
			var denominator uint256.Int
			denominator.Mul(distQ96Base, &q96MinusAlphaPowLength)

			var sqrtRatioTickSpacingMinusQ96 uint256.Int
			sqrtRatioTickSpacingMinusQ96.Sub(sqrtRatioTickSpacing, math.Q96)
			result, err := math.FullMulDivUp(&sqrtRatioTickSpacingMinusQ96, &numerator, &denominator)
			if err != nil {
				return nil, err
			}

			cumulativeAmount1DensityX96 = math.MulDivUp(result, sqrtRatioMinTick, math.Q96)
		}
	}

	return math.FullMulX96Up(cumulativeAmount1DensityX96, totalLiquidity)
}

// InverseCumulativeAmount0 computes the inverse of cumulative amount0
func InverseCumulativeAmount0(
	tickSpacing int,
	cumulativeAmount0_, totalLiquidity *uint256.Int,
	minTick, length int, alphaX96 *uint256.Int,
) (bool, int, error) {
	if cumulativeAmount0_.IsZero() {
		return true, minTick + length*tickSpacing, nil
	}

	cumulativeAmount0DensityX96, err := math.FullMulDivUp(cumulativeAmount0_, math.Q96, totalLiquidity)
	if err != nil {
		return false, 0, err
	}

	sqrtRatioNegTickSpacing, err := math.GetSqrtPriceAtTick(-tickSpacing)
	if err != nil {
		return false, 0, err
	}

	sqrtRatioMinTick, err := math.GetSqrtPriceAtTick(minTick)
	if err != nil {
		return false, 0, err
	}

	baseX96 := math.MulDiv(alphaX96, sqrtRatioNegTickSpacing, math.Q96)

	baseX96Int256 := i256.SafeToInt256(baseX96)

	lnBaseX96, err := math.LnQ96(baseX96Int256)
	if err != nil {
		return false, 0, err
	}

	var xWad *int256.Int
	if alphaX96.Gt(math.Q96) {
		alphaInvX96 := math.MulDiv(math.Q96, math.Q96, alphaX96)

		alphaInvPowLengthX96, err := math.Rpow(alphaInvX96, length, math.Q96)
		if err != nil {
			return false, 0, err
		}

		intermediateTermIsPositive := alphaInvX96.Gt(sqrtRatioNegTickSpacing)

		denominator1 := new(uint256.Int).Sub(math.Q96, sqrtRatioNegTickSpacing)
		term1 := math.MulDivUp(cumulativeAmount0DensityX96, sqrtRatioMinTick, denominator1)

		numerator2 := new(uint256.Int).Sub(math.Q96, alphaInvPowLengthX96)
		term2 := math.MulDivUp(term1, numerator2, math.Q96)

		var numerator3 *uint256.Int
		if intermediateTermIsPositive {
			numerator3 = new(uint256.Int).Sub(alphaInvX96, sqrtRatioNegTickSpacing)
		} else {
			numerator3 = new(uint256.Int).Sub(sqrtRatioNegTickSpacing, alphaInvX96)
		}
		denominator3 := new(uint256.Int).Sub(math.Q96, alphaInvX96)
		tmp := math.MulDivUp(term2, numerator3, denominator3)

		sqrtPriceNegTickSpacingMulLength, err := math.GetSqrtPriceAtTick(-tickSpacing * length)
		if err != nil {
			return false, 0, err
		}

		if !intermediateTermIsPositive && sqrtPriceNegTickSpacingMulLength.Cmp(tmp) <= 0 {
			result := minTick + (length-1)*tickSpacing
			maxCumAmount0, err := CumulativeAmount0(tickSpacing, result, totalLiquidity, minTick, length, alphaX96)
			if err != nil {
				return false, 0, err
			}
			if cumulativeAmount0_.Cmp(maxCumAmount0) <= 0 {
				return true, result, nil
			} else {
				return false, 0, nil
			}
		}

		if intermediateTermIsPositive {
			tmp = new(uint256.Int).Add(tmp, sqrtPriceNegTickSpacingMulLength)
		} else {
			tmp = new(uint256.Int).Sub(sqrtPriceNegTickSpacingMulLength, tmp)
		}

		lnTmpRoundingUp, err := math.LnQ96RoundingUp(i256.SafeToInt256(tmp))
		if err != nil {
			return false, 0, err
		}

		lnAlphaX96RoundingUp, err := math.LnQ96RoundingUp(i256.SafeToInt256(alphaX96))
		if err != nil {
			return false, 0, err
		}

		lengthTimesLnAlpha := new(int256.Int).Mul(int256.NewInt(int64(length)), lnAlphaX96RoundingUp)

		numeratorXWad := lengthTimesLnAlpha.Add(lnTmpRoundingUp, lengthTimesLnAlpha)

		xWad, err = math.SDivWad(numeratorXWad, lnBaseX96)
		if err != nil {
			return false, 0, err
		}
	} else {
		// alpha <= 1
		alphaLengthPow, err := math.Rpow(alphaX96, length, math.Q96)
		if err != nil {
			return false, 0, err
		}

		baseLengthPow, err := math.Rpow(baseX96, length, math.Q96)
		if err != nil {
			return false, 0, err
		}

		denominator := new(uint256.Int).Sub(math.Q96, alphaLengthPow)
		denominator.Mul(denominator, new(uint256.Int).Sub(math.Q96, baseX96))

		term1Num := math.MulDivUp(cumulativeAmount0DensityX96, sqrtRatioMinTick, math.Q96)

		denomTerm := new(uint256.Int).Sub(math.Q96, sqrtRatioNegTickSpacing)
		numerator, err := math.FullMulDivUp(term1Num, denominator, denomTerm)
		if err != nil {
			return false, 0, err
		}

		alphaDenom := new(uint256.Int).Sub(math.Q96, alphaX96)
		quotient := alphaDenom.Div(numerator, alphaDenom)
		basePowXX96 := quotient.Add(quotient, baseLengthPow)

		lnBasePowXX96RoundingUp, err := math.LnQ96RoundingUp(i256.SafeToInt256(basePowXX96))
		if err != nil {
			return false, 0, err
		}

		xWad, err = math.SDivWad(lnBasePowXX96RoundingUp, lnBaseX96)
		if err != nil {
			return false, 0, err
		}
	}

	if xWad.Sign() < 0 {
		maxCumulativeAmount0, err := CumulativeAmount0(tickSpacing, minTick, totalLiquidity, minTick, length, alphaX96)
		if err != nil {
			return false, 0, err
		}
		if cumulativeAmount0_.Gt(maxCumulativeAmount0) {
			return false, 0, nil
		} else {
			xWad = int256.NewInt(0)
		}
	}

	roundedTick := math.XWadToRoundedTick(xWad, minTick, tickSpacing, false)

	maxTick := minTick + length*tickSpacing
	if roundedTick < minTick || roundedTick > maxTick {
		return false, 0, nil
	}

	if roundedTick == maxTick {
		return true, maxTick - tickSpacing, nil
	}

	return true, roundedTick, nil
}

// InverseCumulativeAmount1 computes the inverse of cumulative amount1
func InverseCumulativeAmount1(
	tickSpacing int,
	cumulativeAmount1_, totalLiquidity *uint256.Int,
	minTick, length int, alphaX96 *uint256.Int,
) (bool, int, error) {
	if cumulativeAmount1_.IsZero() {
		return true, minTick - tickSpacing, nil
	}

	cumulativeAmount1DensityX96, err := math.FullMulDiv(cumulativeAmount1_, math.Q96, totalLiquidity)
	if err != nil {
		return false, 0, err
	}

	sqrtRatioTickSpacing, err := math.GetSqrtPriceAtTick(tickSpacing)
	if err != nil {
		return false, 0, err
	}

	baseX96 := math.MulDiv(alphaX96, sqrtRatioTickSpacing, math.Q96)

	lnBaseX96, err := math.LnQ96RoundingUp(i256.SafeToInt256(baseX96))
	if err != nil {
		return false, 0, err
	}

	var xWad *int256.Int
	if alphaX96.Gt(math.Q96) {
		alphaInvX96 := math.MulDiv(math.Q96, math.Q96, alphaX96)

		alphaInvPowLengthX96, err := math.Rpow(alphaInvX96, length, math.Q96)
		if err != nil {
			return false, 0, err
		}

		sqrtRatioNegMinTick, err := math.GetSqrtPriceAtTick(-minTick)
		if err != nil {
			return false, 0, err
		}

		numerator1 := new(uint256.Int).Sub(alphaX96, math.Q96)
		denominator1 := new(uint256.Int).Sub(baseX96, math.Q96)
		denominator2 := new(uint256.Int).Sub(math.Q96, alphaInvPowLengthX96)

		tickSpacingMinusQ96 := new(uint256.Int).Sub(sqrtRatioTickSpacing, math.Q96)
		term1 := math.MulDiv(cumulativeAmount1DensityX96, sqrtRatioNegMinTick, tickSpacingMinusQ96)
		term2 := math.MulDiv(term1, denominator1, numerator1)
		numerator2 := math.MulDiv(term2, denominator2, math.Q96)

		sumForCheck := new(uint256.Int).Add(numerator2, alphaInvPowLengthX96)
		if sumForCheck.IsZero() {
			return false, 0, nil
		}

		lnSum, err := math.LnQ96(i256.SafeToInt256(sumForCheck))
		if err != nil {
			return false, 0, err
		}

		lnAlphaX96, err := math.LnQ96(i256.SafeToInt256(alphaX96))
		if err != nil {
			return false, 0, err
		}

		lengthInt256 := int256.NewInt(int64(length))
		lengthTimesLnAlpha := lengthInt256.Mul(lengthInt256, lnAlphaX96)

		numeratorXWad := lengthTimesLnAlpha.Add(lnSum, lengthTimesLnAlpha)

		xWadBeforeSub, err := math.SDivWad(numeratorXWad, lnBaseX96)
		if err != nil {
			return false, 0, err
		}

		xWad = xWadBeforeSub.Sub(xWadBeforeSub, math.WAD_INT)

	} else {
		sqrtRatioMinTick, err := math.GetSqrtPriceAtTick(minTick)
		if err != nil {
			return false, 0, err
		}

		distQ96Base := math.Dist(math.Q96, baseX96)
		alphaPowLength, err := math.Rpow(alphaX96, length, math.Q96)
		if err != nil {
			return false, 0, err
		}
		term2Denom := new(uint256.Int).Sub(math.Q96, alphaPowLength)
		denominator := new(uint256.Int).Mul(distQ96Base, term2Denom)

		term1Num, err := math.FullMulDiv(cumulativeAmount1DensityX96, math.Q96, sqrtRatioMinTick)
		if err != nil {
			return false, 0, err
		}

		tickSpacingMinusQ96 := new(uint256.Int).Sub(sqrtRatioTickSpacing, math.Q96)
		numerator, err := math.FullMulDiv(term1Num, denominator, tickSpacingMinusQ96)
		if err != nil {
			return false, 0, err
		}

		if math.Q96.Gt(baseX96) {
			alphaDenom := new(uint256.Int).Sub(math.Q96, alphaX96)
			quotient := new(uint256.Int).Div(numerator, alphaDenom)
			if math.Q96.Cmp(quotient) <= 0 {
				minTickCumAmount1, err := CumulativeAmount1(tickSpacing, minTick, totalLiquidity, minTick, length, alphaX96)
				if err != nil {
					return false, 0, err
				}
				if cumulativeAmount1_.Cmp(minTickCumAmount1) <= 0 {
					return true, minTick, nil
				} else {
					return false, 0, nil
				}
			}
		}

		alphaDenom := new(uint256.Int).Sub(math.Q96, alphaX96)
		quotient := new(uint256.Int).Div(numerator, alphaDenom)
		var basePowXPlusOneX96 *uint256.Int
		if math.Q96.Gt(baseX96) {
			basePowXPlusOneX96 = new(uint256.Int).Sub(math.Q96, quotient)
		} else {
			basePowXPlusOneX96 = new(uint256.Int).Add(math.Q96, quotient)
		}

		lnBasePow, err := math.LnQ96(i256.SafeToInt256(basePowXPlusOneX96))
		if err != nil {
			return false, 0, err
		}

		xWadBeforeSub, err := math.SDivWad(lnBasePow, lnBaseX96)
		if err != nil {
			return false, 0, err
		}

		xWad = xWadBeforeSub.Sub(xWadBeforeSub, math.WAD_INT)
	}

	xWadMax := int256.NewInt(int64(length - 1))
	xWadMax.Mul(xWadMax, math.WAD_INT)

	if xWad.Gt(xWadMax) {
		maxTick := minTick + (length-1)*tickSpacing
		maxCumulativeAmount1, err := CumulativeAmount1(tickSpacing, maxTick, totalLiquidity, minTick, length, alphaX96)
		if err != nil {
			return false, 0, err
		}
		if cumulativeAmount1_.Gt(maxCumulativeAmount1) {
			return false, 0, nil
		} else {
			xWad = xWadMax
		}
	}

	roundedTick := math.XWadToRoundedTick(xWad, minTick, tickSpacing, true)

	if roundedTick < minTick-tickSpacing || roundedTick >= minTick+length*tickSpacing {
		return false, 0, nil
	}

	if roundedTick == minTick-tickSpacing {
		return true, minTick, nil
	}

	return true, roundedTick, nil
}
