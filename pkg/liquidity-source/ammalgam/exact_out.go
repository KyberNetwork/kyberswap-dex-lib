package ammalgam

import (
	"errors"

	"github.com/holiman/uint256"
)

var errExceededExpectedIterations = errors.New("ammalgam exact-out exceeded expected iterations")

var (
	maxReserve112                  = uint256.MustFromHex("0xffffffffffffffffffffffffffff")
	q64                            = uint256.MustFromHex("0x10000000000000000")
	bips                           = uint256.NewInt(10_000)
	nTimes100                      = uint256.NewInt(2000)
	nQ64                           = uint256.MustFromHex("0x140000000000000000")
	nQ64Times100                   = uint256.MustFromHex("0x7d00000000000000000")
	minKeepQ64                     = new(uint256.Int).Sub(bipsQ64, minFeeQ64)
	maxKeepQ64                     = uint256.MustFromHex("0x7d00000000000000000") // 20 * 100 * Q64
	linearOutDelta                 = uint256.MustFromHex("0x3200000000000000000") // 800 * Q64
	maxFeeSquaredQ64               = uint256.MustFromHex("0x6400000000000000000") // 1600 * Q64
	boundaryNumerator              = uint256.NewInt(2400)
	boundaryDenominator            = uint256.NewInt(4400)
	discriminantFactor             = uint256.NewInt(8000)
	crossingQuadraticDivisor       = uint256.NewInt(4000)
	crossingDiscriminantBaseFactor = uint256.NewInt(12500)
	maxFeeMultiplier               = uint256.NewInt(5)
	maxFeeCorrection               = uint256.NewInt(4)
	quadraticInnerBaseQ64          = uint256.MustFromHex("0x320000000000000000") // 50 * Q64
	maxSafeForSquare               = uint256.MustFromHex("0xffffffffffffffffffffffffffffffff")
)

type exactOutPool struct {
	reserveIn, referenceReserveIn, reserveOut, missingIn, missingOut uint256.Int
}

func gte(x, y *uint256.Int) bool { return x.Gt(y) || x.Eq(y) }

func lte(x, y *uint256.Int) bool { return x.Lt(y) || x.Eq(y) }

func quoteExactOut(amountOut, reserveIn, referenceReserveIn, reserveOut, missingIn, missingOut *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	if missingOut.Gt(reserveOut) || missingOut.Eq(reserveOut) {
		return zero, errMissingOutGteReserveOut
	}
	available, err := subChecked(reserveOut, missingOut)
	if err != nil {
		return zero, err
	}
	if amountOut.Gt(&available) || amountOut.Eq(&available) {
		return zero, errSwapAmountOutExceedsAvailable
	}
	if amountOut.IsZero() {
		return zero, nil
	}
	p := exactOutPool{*reserveIn, *referenceReserveIn, *reserveOut, *missingIn, *missingOut}
	input, err := computeExpectedSwapInAmount(amountOut, &p)
	if err != nil {
		return zero, err
	}
	maxInput, err := subChecked(maxReserve112, reserveIn)
	if err != nil {
		return zero, err
	}
	if input.Gt(&maxInput) {
		return zero, errMathOverflow
	}
	return input, nil
}

func computeExpectedSwapInAmount(target *uint256.Int, p *exactOutPool) (uint256.Int, error) {
	var zero uint256.Int
	minimumReserveInBipsQ64, err := invertCurveBipsQ64(target, &p.reserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
	if err != nil {
		return zero, err
	}
	reserveBips, err := mulChecked(&p.reserveIn, bipsQ64)
	if err != nil {
		return zero, err
	}
	swapInAfterFee, err := subChecked(&minimumReserveInBipsQ64, &reserveBips)
	if err != nil {
		return zero, err
	}
	minimumSwapIn, err := ceilDiv(&swapInAfterFee, minKeepQ64)
	if err != nil {
		return zero, err
	}
	out, err := quoteExactIn(&minimumSwapIn, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
	if err != nil {
		return zero, err
	}
	if gte(&out, target) {
		return minimumSwapIn, nil
	}
	minimumWithAllowance, err := addChecked(&minimumSwapIn, maxFeeMultiplier)
	if err != nil {
		return zero, err
	}
	out, err = quoteExactIn(&minimumWithAllowance, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
	if err != nil {
		return zero, err
	}
	if gte(&out, target) {
		low, err := addChecked(&minimumSwapIn, one)
		if err != nil {
			return zero, err
		}
		return exactOutBinarySearch(&low, &minimumWithAllowance, target, p)
	}
	maxCost, err := ceilDiv(&swapInAfterFee, maxKeepQ64)
	if err != nil {
		return zero, err
	}
	maximumSwapIn, err := addChecked(&maxCost, maxFeeMultiplier)
	if err != nil {
		return zero, err
	}
	var estimate uint256.Int
	if p.referenceReserveIn.Gt(&p.reserveIn) {
		estimate, err = estimateCrossing(&swapInAfterFee, &p.reserveIn, &p.referenceReserveIn)
	} else {
		estimate, err = estimateNonCrossing(target, p)
	}
	if err != nil {
		return zero, err
	}
	return refineEstimate(&estimate, &minimumSwapIn, &maximumSwapIn, target, p)
}

func invertCurveBipsQ64(target, reserveIn, reserveOut, missingIn, missingOut *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	targetReserveOut, err := subChecked(reserveOut, target)
	if err != nil {
		return zero, err
	}
	adjustedReserveIn, err := actualToAdjusted(reserveIn, missingIn)
	if err != nil {
		return zero, err
	}
	adjustedReserveOut, err := actualToAdjusted(reserveOut, missingOut)
	if err != nil {
		return zero, err
	}
	k, err := mulChecked(&adjustedReserveIn, &adjustedReserveOut)
	if err != nil {
		return zero, err
	}
	left, err := mulChecked(missingOut, bufferNumerator)
	if err != nil {
		return zero, err
	}
	right, err := mulChecked(&targetReserveOut, buffer)
	if err != nil {
		return zero, err
	}
	var maxAdjustedTargetOut uint256.Int
	if left.Lt(&right) {
		maxAdjustedTargetOut = targetReserveOut
	} else {
		remaining, err := subChecked(&targetReserveOut, missingOut)
		if err != nil {
			return zero, err
		}
		maxAdjustedTargetOut, err = mulChecked(&remaining, bufferNumerator)
		if err != nil {
			return zero, err
		}
	}
	minAdjustedReserveIn, err := ceilDiv(&k, &maxAdjustedTargetOut)
	if err != nil {
		return zero, err
	}
	oneLess, err := subChecked(&minAdjustedReserveIn, one)
	if err != nil {
		return zero, err
	}
	minimumReserveInNonDepleted, err := mulChecked(&oneLess, bipsQ64)
	if err != nil {
		return zero, err
	}
	minimumReserveInNonDepleted, err = addChecked(&minimumReserveInNonDepleted, one)
	if err != nil {
		return zero, err
	}
	left, err = mulChecked(missingIn, bipsQ64)
	if err != nil {
		return zero, err
	}
	left, err = mulChecked(&left, bufferNumerator)
	if err != nil {
		return zero, err
	}
	right, err = mulChecked(&minimumReserveInNonDepleted, buffer)
	if err != nil {
		return zero, err
	}
	if left.Lt(&right) {
		return minimumReserveInNonDepleted, nil
	}
	base, err := mulChecked(missingIn, bipsQ64)
	if err != nil {
		return zero, err
	}
	minimumPlusOne, err := addChecked(&minimumReserveInNonDepleted, one)
	if err != nil {
		return zero, err
	}
	correction, err := ceilDiv(&minimumPlusOne, bufferNumerator)
	if err != nil {
		return zero, err
	}
	return addChecked(&base, &correction)
}

func estimateCrossing(costQ64, reserveIn, reference *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	costBips, err := ceilDiv(costQ64, q64)
	if err != nil {
		return zero, err
	}
	reserveDiff, err := subChecked(reference, reserveIn)
	if err != nil {
		return zero, err
	}
	negB, err := mulChecked(bips, reference)
	if err != nil {
		return zero, err
	}
	tmp, err := mulChecked(crossingQuadraticDivisor, &reserveDiff)
	if err != nil {
		return zero, err
	}
	negB, err = addChecked(&negB, &tmp)
	if err != nil {
		return zero, err
	}
	discriminantBase, err := mulChecked(crossingDiscriminantBaseFactor, reference)
	if err != nil {
		return zero, err
	}
	tmp, err = mulChecked(bips, &reserveDiff)
	if err != nil {
		return zero, err
	}
	discriminantBase, err = addChecked(&discriminantBase, &tmp)
	if err != nil {
		return zero, err
	}
	positive := gte(&discriminantBase, &costBips)
	inner, err := subChecked(&discriminantBase, &costBips)
	if !positive {
		inner, err = subChecked(&costBips, &discriminantBase)
	}
	if err != nil {
		return zero, err
	}
	factor, err := mulChecked(discriminantFactor, reference)
	discOverflow := err != nil
	var estimate uint256.Int
	if !discOverflow {
		maxQuotient := new(uint256.Int).Div(maxUint256, &factor)
		discOverflow = inner.Gt(maxQuotient)
	}
	if !discOverflow {
		discriminant, err := mulChecked(&factor, &inner)
		if err != nil {
			return zero, err
		}
		var sqrt uint256.Int
		sqrt.Sqrt(&discriminant)
		if positive && lte(&sqrt, &negB) {
			negBSquared, err := mulChecked(&negB, &negB)
			if err != nil {
				return zero, err
			}
			numerator, err := subChecked(&negBSquared, &discriminant)
			if err != nil {
				return zero, err
			}
			sum, err := addChecked(&negB, &sqrt)
			if err != nil {
				return zero, err
			}
			denominator, err := mulChecked(crossingQuadraticDivisor, &sum)
			if err != nil {
				return zero, err
			}
			estimate, err = ceilDiv(&numerator, &denominator)
			if err != nil {
				return zero, err
			}
		} else if !positive {
			numerator, err := addChecked(&negB, &sqrt)
			if err != nil {
				return zero, err
			}
			estimate, err = ceilDiv(&numerator, crossingQuadraticDivisor)
			if err != nil {
				return zero, err
			}
		}
	}
	twiceReference, err := mulChecked(two, reference)
	if err != nil {
		return zero, err
	}
	limit, err := addChecked(&reserveDiff, &twiceReference)
	if err != nil {
		return zero, err
	}
	fallback := estimate.IsZero() || lte(&estimate, &reserveDiff) || estimate.Gt(&limit)
	if fallback {
		linearBase, err := ceilDiv(&costBips, nTimes100)
		if err != nil {
			return zero, err
		}
		maxFeeBase, err := subChecked(&twiceReference, reserveIn)
		if err != nil {
			return zero, err
		}
		maxFeeOffset, err := mulChecked(maxFeeCorrection, &maxFeeBase)
		if err != nil {
			return zero, err
		}
		if linearBase.Gt(&maxFeeOffset) {
			return subChecked(&linearBase, &maxFeeOffset)
		}
		return addChecked(&limit, one)
	}
	return estimate, nil
}

func estimateNonCrossing(target *uint256.Int, p *exactOutPool) (uint256.Int, error) {
	var zero uint256.Int
	adjReserveOut := p.reserveOut
	left, err := mulChecked(&p.reserveOut, buffer)
	if err != nil {
		return zero, err
	}
	right, err := mulChecked(bufferNumerator, &p.missingOut)
	if err != nil {
		return zero, err
	}
	if left.Lt(&right) {
		remaining, err := subChecked(&p.reserveOut, &p.missingOut)
		if err != nil {
			return zero, err
		}
		numerator, err := mulChecked(&remaining, bufferNumerator)
		if err != nil {
			return zero, err
		}
		adjReserveOut, err = divChecked(&numerator, one)
		if err != nil {
			return zero, err
		}
	}
	adjReserveIn := p.reserveIn
	left, err = mulChecked(&p.reserveIn, buffer)
	if err != nil {
		return zero, err
	}
	right, err = mulChecked(bufferNumerator, &p.missingIn)
	if err != nil {
		return zero, err
	}
	if left.Lt(&right) {
		remaining, err := subChecked(&p.reserveIn, &p.missingIn)
		if err != nil {
			return zero, err
		}
		numerator, err := mulChecked(&remaining, bufferNumerator)
		if err != nil {
			return zero, err
		}
		adjReserveIn, err = divChecked(&numerator, one)
		if err != nil {
			return zero, err
		}
	}
	adjSwapAmountOut := *target
	targetReserveOut, err := subChecked(&p.reserveOut, target)
	if err != nil {
		return zero, err
	}
	left, err = mulChecked(&targetReserveOut, buffer)
	if err != nil {
		return zero, err
	}
	right, err = mulChecked(bufferNumerator, &p.missingOut)
	if err != nil {
		return zero, err
	}
	if left.Lt(&right) {
		remaining, err := subChecked(&targetReserveOut, &p.missingOut)
		if err != nil {
			return zero, err
		}
		numerator, err := mulChecked(bufferNumerator, &remaining)
		if err != nil {
			return zero, err
		}
		correction, err := divChecked(&numerator, one)
		if err != nil {
			return zero, err
		}
		adjSwapAmountOut, err = subChecked(&adjReserveOut, &correction)
		if err != nil {
			return zero, err
		}
	}
	if adjReserveIn.Lt(&p.referenceReserveIn) {
		return fallbackEstimate(&adjReserveIn, &adjReserveOut, &adjSwapAmountOut, &p.referenceReserveIn)
	}
	if p.reserveIn.Eq(&p.referenceReserveIn) {
		maxInput, err := calculateMaxSwapAmountInWithMinimumFee(&p.referenceReserveIn)
		if err != nil {
			return zero, err
		}
		maxOut, err := quoteExactIn(&maxInput, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
		if err != nil {
			return zero, err
		}
		if lte(target, &maxOut) {
			return calculateSwapAmountInMinimumFee(&adjSwapAmountOut, &adjReserveIn, &adjReserveOut)
		}
	}
	var useLinear bool
	twiceReference, err := mulChecked(&p.referenceReserveIn, two)
	if err != nil {
		return zero, err
	}
	if gte(&adjReserveIn, &twiceReference) {
		useLinear = true
	} else if !p.reserveIn.Eq(&p.referenceReserveIn) && p.reserveIn.Lt(&twiceReference) {
		twiceReference, err := mulChecked(&p.referenceReserveIn, two)
		if err != nil {
			return zero, err
		}
		boundaryBase, err := subChecked(&twiceReference, &p.reserveIn)
		if err != nil {
			return zero, err
		}
		boundaryInput, err := mulChecked(two, &boundaryBase)
		if err != nil {
			return zero, err
		}
		maxOut, err := quoteExactIn(&boundaryInput, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
		if err != nil {
			return zero, err
		}
		useLinear = target.Gt(&maxOut)
	} else {
		boundaryProduct, err := mulChecked(boundaryNumerator, &adjReserveOut)
		if err != nil {
			return zero, err
		}
		boundary, err := divChecked(&boundaryProduct, boundaryDenominator)
		if err != nil {
			return zero, err
		}
		useLinear = adjSwapAmountOut.Gt(&boundary)
	}
	if useLinear {
		overflow, err := wouldOverflowLinear(&adjReserveIn, &p.referenceReserveIn, &adjSwapAmountOut, &adjReserveOut)
		if err != nil {
			return zero, err
		}
		if overflow {
			return fallbackEstimate(&adjReserveIn, &adjReserveOut, &adjSwapAmountOut, &p.referenceReserveIn)
		}
		return calculateSwapAmountInLinearFee(&adjSwapAmountOut, &adjReserveIn, &p.referenceReserveIn, &adjReserveOut)
	}
	overflow, err := wouldOverflowQuadratic(&adjReserveIn, &p.referenceReserveIn, &adjSwapAmountOut, &adjReserveOut)
	if err != nil {
		return zero, err
	}
	if overflow {
		return fallbackEstimate(&adjReserveIn, &adjReserveOut, &adjSwapAmountOut, &p.referenceReserveIn)
	}
	return calculateSwapAmountInQuadraticFee(&adjSwapAmountOut, &adjReserveIn, &p.referenceReserveIn, &adjReserveOut)
}

func calculateSwapAmountInMinimumFee(amountOut, reserveIn, reserveOut *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	feeAdjustedReserve, err := mulChecked(bipsQ64, reserveIn)
	if err != nil {
		return zero, err
	}
	remaining, err := subChecked(reserveOut, amountOut)
	if err != nil {
		return zero, err
	}
	denominator, err := mulChecked(minKeepQ64, &remaining)
	if err != nil {
		return zero, err
	}
	return mulDivCeil(&feeAdjustedReserve, amountOut, &denominator)
}

// mulDivCeil mirrors OpenZeppelin Math.mulDiv(..., Math.Rounding.Ceil): the
// multiplication is evaluated at 512-bit precision and only the quotient must
// fit uint256.
func mulDivCeil(x, y, denominator *uint256.Int) (uint256.Int, error) {
	var result uint256.Int
	if denominator.IsZero() {
		return result, errMathDivisionByZero
	}
	if _, overflow := result.MulDivOverflow(x, y, denominator); overflow {
		return uint256.Int{}, errMathOverflow
	}
	var remainder uint256.Int
	remainder.MulMod(x, y, denominator)
	if !remainder.IsZero() {
		return addChecked(&result, one)
	}
	return result, nil
}

func calculateMaxSwapAmountInWithMinimumFee(reference *uint256.Int) (uint256.Int, error) {
	numerator, err := mulChecked(reference, minFeeQ64)
	if err != nil {
		return uint256.Int{}, err
	}
	return ceilDiv(&numerator, nQ64Times100)
}

func fallbackEstimate(reserveIn, reserveOut, amountOut, reference *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	denominator, err := subChecked(reserveOut, amountOut)
	if err != nil {
		return zero, err
	}
	numerator, err := mulChecked(maxFeeMultiplier, reserveIn)
	if err != nil {
		return zero, err
	}
	numerator, err = mulChecked(&numerator, amountOut)
	if err != nil {
		return zero, err
	}
	newReserveIn, err := ceilDiv(&numerator, &denominator)
	if err != nil {
		return zero, err
	}
	correction, err := mulChecked(maxFeeCorrection, reference)
	if err != nil {
		return zero, err
	}
	if newReserveIn.Gt(&correction) {
		return subChecked(&newReserveIn, &correction)
	}
	return *one, nil
}

func wouldOverflowQuadratic(currentReserveIn, referenceReserveIn, amountOut, currentReserveOut *uint256.Int) (bool, error) {
	denom, err := subChecked(currentReserveOut, amountOut)
	if err != nil {
		return false, err
	}
	if denom.IsZero() {
		return true, nil
	}
	constNQ128 := new(uint256.Int).Lsh(nTimes100, 128)
	mul, err := mulChecked(constNQ128, currentReserveIn)
	if err != nil {
		return false, err
	}
	mulDivResult, err := ceilDiv(&mul, referenceReserveIn)
	if err != nil {
		return false, err
	}
	outPlusOne, err := addChecked(amountOut, one)
	if err != nil {
		return false, err
	}
	limit := new(uint256.Int).Div(maxUint256, &outPlusOne)
	if mulDivResult.Gt(limit) {
		return true, nil
	}
	sqrtInput, err := mulChecked(&mulDivResult, amountOut)
	if err != nil {
		return false, err
	}
	sqrtInput, err = divChecked(&sqrtInput, &denom)
	if err != nil {
		return false, err
	}
	reserveDiff, err := subChecked(currentReserveIn, referenceReserveIn)
	if err != nil {
		return false, err
	}
	inner, err := mulChecked(twenty, &reserveDiff)
	if err != nil {
		return false, err
	}
	inner, err = mulChecked(&inner, q64)
	if err != nil {
		return false, err
	}
	inner, err = ceilDiv(&inner, referenceReserveIn)
	if err != nil {
		return false, err
	}
	inner, err = subChecked(quadraticInnerBaseQ64, &inner)
	if err != nil {
		return false, err
	}
	square, err := mulChecked(&inner, &inner)
	if err != nil {
		return false, err
	}
	return square.Lt(&sqrtInput), nil
}

func wouldOverflowLinear(currentReserveIn, referenceReserveIn, amountOut, currentReserveOut *uint256.Int) (bool, error) {
	denom, err := subChecked(currentReserveOut, amountOut)
	if err != nil {
		return false, err
	}
	if denom.IsZero() {
		return true, nil
	}
	product, err := mulChecked(nQ64Times100, currentReserveIn)
	if err != nil {
		return false, err
	}
	saom, err := divChecked(&product, referenceReserveIn)
	if err != nil {
		return false, err
	}
	outPlusOne, err := addChecked(amountOut, one)
	if err != nil {
		return false, err
	}
	limit := new(uint256.Int).Div(maxUint256, &outPlusOne)
	if saom.Gt(limit) {
		return true, nil
	}
	scaled, err := mulChecked(&saom, amountOut)
	if err != nil {
		return false, err
	}
	scaled, err = divChecked(&scaled, &denom)
	if err != nil {
		return false, err
	}
	return scaled.Gt(maxSafeForSquare), nil
}

func calculateSwapAmountInQuadraticFee(amountOut, currentReserveIn, referenceReserveIn, currentReserveOut *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	reserveDiff, err := subChecked(currentReserveIn, referenceReserveIn)
	if err != nil {
		return zero, err
	}
	inner, err := mulChecked(twenty, &reserveDiff)
	if err != nil {
		return zero, err
	}
	inner, err = mulChecked(&inner, q64)
	if err != nil {
		return zero, err
	}
	inner, err = ceilDiv(&inner, referenceReserveIn)
	if err != nil {
		return zero, err
	}
	inner, err = subChecked(quadraticInnerBaseQ64, &inner)
	if err != nil {
		return zero, err
	}
	constant, err := mulChecked(nTimes100, new(uint256.Int).Lsh(uint256.NewInt(1), 128))
	if err != nil {
		return zero, err
	}
	firstNumerator, err := mulChecked(&constant, currentReserveIn)
	if err != nil {
		return zero, err
	}
	first, err := ceilDiv(&firstNumerator, referenceReserveIn)
	if err != nil {
		return zero, err
	}
	secondNumerator, err := mulChecked(&first, amountOut)
	if err != nil {
		return zero, err
	}
	denom, err := subChecked(currentReserveOut, amountOut)
	if err != nil {
		return zero, err
	}
	discriminant, err := ceilDiv(&secondNumerator, &denom)
	if err != nil {
		return zero, err
	}
	square, err := mulChecked(&inner, &inner)
	if err != nil {
		return zero, err
	}
	radical, err := subChecked(&square, &discriminant)
	if err != nil {
		return zero, err
	}
	var sqrt uint256.Int
	sqrt.Sqrt(&radical)
	numerator, err := mulChecked(referenceReserveIn, &discriminant)
	if err != nil {
		return zero, err
	}
	denomInner, err := addChecked(&inner, &sqrt)
	if err != nil {
		return zero, err
	}
	denominator, err := mulChecked(nQ64, &denomInner)
	if err != nil {
		return zero, err
	}
	return ceilDiv(&numerator, &denominator)
}

func calculateSwapAmountInLinearFee(amountOut, currentReserveIn, referenceReserveIn, currentReserveOut *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	reserveDiff, err := subChecked(currentReserveIn, referenceReserveIn)
	if err != nil {
		return zero, err
	}
	reserveTerm, err := mulChecked(linearOutDelta, &reserveDiff)
	if err != nil {
		return zero, err
	}
	reserveTerm, err = divChecked(&reserveTerm, referenceReserveIn)
	if err != nil {
		return zero, err
	}
	swapTerm, err := mulChecked(nQ64Times100, currentReserveIn)
	if err != nil {
		return zero, err
	}
	swapTerm, err = divChecked(&swapTerm, referenceReserveIn)
	if err != nil {
		return zero, err
	}
	swapTerm, err = mulChecked(&swapTerm, amountOut)
	if err != nil {
		return zero, err
	}
	denom, err := subChecked(currentReserveOut, amountOut)
	if err != nil {
		return zero, err
	}
	swapTerm, err = divChecked(&swapTerm, &denom)
	if err != nil {
		return zero, err
	}
	reservePlus, err := addChecked(&reserveTerm, maxFeeSquaredQ64)
	if err != nil {
		return zero, err
	}
	swapDominant := reservePlus.Lt(&swapTerm)
	var linearTerm uint256.Int
	if swapDominant {
		linearTerm, err = subChecked(&swapTerm, &reserveTerm)
		if err != nil {
			return zero, err
		}
		linearTerm, err = subChecked(&linearTerm, maxFeeSquaredQ64)
	} else {
		linearTerm, err = subChecked(&reservePlus, &swapTerm)
	}
	if err != nil {
		return zero, err
	}
	linearSquare, err := mulChecked(&linearTerm, &linearTerm)
	if err != nil {
		return zero, err
	}
	discriminantTerm, err := mulChecked(&reserveTerm, &swapTerm)
	if err != nil {
		return zero, err
	}
	discriminantTerm, err = mulChecked(discriminantTermFactor, &discriminantTerm)
	if err != nil {
		return zero, err
	}
	discriminant, err := addChecked(&linearSquare, &discriminantTerm)
	if err != nil {
		return zero, err
	}
	var sqrt uint256.Int
	sqrt.Sqrt(&discriminant)
	if swapDominant {
		sum, err := addChecked(&sqrt, &linearTerm)
		if err != nil {
			return zero, err
		}
		numerator, err := mulChecked(referenceReserveIn, &sum)
		if err != nil {
			return zero, err
		}
		return ceilDiv(&numerator, linearOutDelta)
	}
	numerator, err := mulChecked(uint256.NewInt(4), &reserveDiff)
	if err != nil {
		return zero, err
	}
	numerator, err = mulChecked(&numerator, &swapTerm)
	if err != nil {
		return zero, err
	}
	denominator, err := addChecked(&sqrt, &linearTerm)
	if err != nil {
		return zero, err
	}
	return ceilDiv(&numerator, &denominator)
}

var discriminantTermFactor = uint256.NewInt(4)

func refineEstimate(estimate, minimum, maximum, target *uint256.Int, p *exactOutPool) (uint256.Int, error) {
	var zero uint256.Int
	if maximum.Lt(minimum) {
		maximum = minimum
	}
	if estimate.Lt(minimum) {
		estimate = minimum
	}
	if estimate.Gt(maximum) {
		estimate = maximum
	}
	output, err := quoteExactIn(estimate, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
	if err != nil {
		return zero, err
	}
	if gte(&output, target) {
		if lte(estimate, minimum) {
			return *estimate, nil
		}
		previousInput, err := subChecked(estimate, one)
		if err != nil {
			return zero, err
		}
		previous, err := quoteExactIn(&previousInput, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
		if err != nil {
			return zero, err
		}
		if previous.Lt(target) {
			return *estimate, nil
		}
		low := *minimum
		high, err := subChecked(estimate, one)
		if err != nil {
			return zero, err
		}
		return exponentialThenBinarySearch(&low, &high, estimate, target, p, false)
	}
	low, err := addChecked(estimate, one)
	if err != nil {
		return zero, err
	}
	return exponentialThenBinarySearch(&low, maximum, estimate, target, p, true)
}

func exponentialThenBinarySearch(low, high, estimate, target *uint256.Int, p *exactOutPool, searchUp bool) (uint256.Int, error) {
	var zero uint256.Int
	step := uint256.NewInt(1)
	if searchUp {
		for {
			candidate, err := addChecked(estimate, step)
			if err != nil {
				return zero, err
			}
			if !candidate.Lt(high) {
				break
			}
			output, err := quoteExactIn(&candidate, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
			if err != nil {
				return zero, err
			}
			if gte(&output, target) {
				half := new(uint256.Int).Rsh(step, 1)
				searchLow, err := addChecked(estimate, half)
				if err != nil {
					return zero, err
				}
				searchLow, err = addChecked(&searchLow, one)
				if err != nil {
					return zero, err
				}
				return exactOutBinarySearch(&searchLow, &candidate, target, p)
			}
			nextStep, err := mulChecked(step, two)
			if err != nil {
				return zero, err
			}
			step.Set(&nextStep)
		}
		half := new(uint256.Int).Rsh(step, 1)
		searchLow, err := addChecked(estimate, half)
		if err != nil {
			return zero, err
		}
		searchLow, err = addChecked(&searchLow, one)
		if err != nil {
			return zero, err
		}
		if searchLow.Lt(low) {
			searchLow.Set(low)
		}
		return exactOutBinarySearch(&searchLow, high, target, p)
	}
	for {
		distance, err := subChecked(estimate, low)
		if err != nil {
			return zero, err
		}
		if !step.Lt(&distance) {
			break
		}
		candidate, err := subChecked(estimate, step)
		if err != nil {
			return zero, err
		}
		output, err := quoteExactIn(&candidate, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
		if err != nil {
			return zero, err
		}
		if output.Lt(target) {
			searchLow, err := addChecked(&candidate, one)
			if err != nil {
				return zero, err
			}
			half := new(uint256.Int).Rsh(step, 1)
			searchHigh, err := subChecked(estimate, half)
			if err != nil {
				return zero, err
			}
			return exactOutBinarySearch(&searchLow, &searchHigh, target, p)
		}
		nextStep, err := mulChecked(step, two)
		if err != nil {
			return zero, err
		}
		step.Set(&nextStep)
	}
	half := new(uint256.Int).Rsh(step, 1)
	searchHigh, err := subChecked(estimate, half)
	if err != nil {
		return zero, err
	}
	if searchHigh.Gt(high) {
		searchHigh.Set(high)
	}
	return exactOutBinarySearch(low, &searchHigh, target, p)
}

func exactOutBinarySearch(low, high, target *uint256.Int, p *exactOutPool) (uint256.Int, error) {
	var zero uint256.Int
	distance, err := subChecked(high, low)
	if err != nil {
		return zero, err
	}
	// Core caps the wide search at 60 based on its expected estimator error.
	// Exact-output is an off-chain routing facility here, and executable pair
	// reserves are uint112, so 112 iterations is the actual hard upper bound.
	// Keeping that bound avoids rejecting valid exact-in executions when a
	// heavily depleted pool's estimate is farther away than core anticipated.
	maxIterations := uint64(112)
	if distance.Cmp(uint256.NewInt(1<<20)) <= 0 {
		maxIterations = 20
	}
	for rounds := uint64(0); low.Lt(high); rounds++ {
		if rounds >= maxIterations {
			return zero, errExceededExpectedIterations
		}
		var sum, midpoint uint256.Int
		if _, overflow := sum.AddOverflow(low, high); overflow {
			return zero, errMathOverflow
		}
		midpoint.Rsh(&sum, 1)
		output, err := quoteExactIn(&midpoint, &p.reserveIn, &p.referenceReserveIn, &p.reserveOut, &p.missingIn, &p.missingOut)
		if err != nil {
			return zero, err
		}
		if gte(&output, target) {
			high.Set(&midpoint)
		} else {
			low.AddUint64(&midpoint, 1)
		}
	}
	return *low, nil
}
