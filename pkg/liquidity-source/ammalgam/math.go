package ammalgam

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	errMathOverflow                  = errors.New("ammalgam math overflow")
	errMathDivisionByZero            = errors.New("ammalgam math division by zero")
	errMissingGteActual              = errors.New("missing assets greater than or equal to actual assets")
	errMissingOutGteReserveOut       = errors.New("missing output assets greater than or equal to output reserve")
	errSwapAmountOutExceedsAvailable = errors.New("swap amount out exceeds available")
)

var (
	bipsQ64                   = uint256.MustFromHex("0x27100000000000000000")
	minFeeQ64                 = uint256.MustFromHex("0x1999999999999999")
	feeScaleQ64               = uint256.MustFromHex("0x7d00000000000000000") // N * 100 * Q64
	maxQuadraticFeeQ64        = uint256.MustFromHex("0x280000000000000000")
	twoQ64                    = uint256.MustFromHex("0x20000000000000000")
	buffer                    = uint256.NewInt(19)
	bufferNumerator           = uint256.NewInt(20)
	linearStartReferenceScale = uint256.NewInt(4)
	two                       = uint256.NewInt(2)
	fourThousand              = uint256.NewInt(4000)
	twenty                    = uint256.NewInt(20)
	one                       = uint256.NewInt(1)
	maxUint256                = new(uint256.Int).SetAllOne()
)

func addChecked(x, y *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.AddOverflow(x, y); overflow {
		return uint256.Int{}, errMathOverflow
	}
	return z, nil
}

func subChecked(x, y *uint256.Int) (uint256.Int, error) {
	if x.Lt(y) {
		return uint256.Int{}, errMathOverflow
	}
	var z uint256.Int
	z.Sub(x, y)
	return z, nil
}

func mulChecked(x, y *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.MulOverflow(x, y); overflow {
		return uint256.Int{}, errMathOverflow
	}
	return z, nil
}

func divChecked(x, y *uint256.Int) (uint256.Int, error) {
	if y.IsZero() {
		return uint256.Int{}, errMathDivisionByZero
	}
	var z uint256.Int
	z.Div(x, y)
	return z, nil
}

func ceilDiv(x, y *uint256.Int) (uint256.Int, error) {
	if y.IsZero() {
		return uint256.Int{}, errMathDivisionByZero
	}
	var q, r uint256.Int
	q.DivMod(x, y, &r)
	if !r.IsZero() {
		var err error
		q, err = addChecked(&q, one)
		if err != nil {
			return uint256.Int{}, err
		}
	}
	return q, nil
}

// calculateSwapFeeBipsQ64 mirrors QuadraticSwapFees.calculateSwapFeeBipsQ64.
func calculateSwapFeeBipsQ64(input, currentReserve, referenceReserve *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	if input.IsZero() {
		return zero, nil
	}
	currentAfter, err := addChecked(input, currentReserve)
	if err != nil {
		return zero, err
	}
	var fee uint256.Int
	if currentReserve.Gt(referenceReserve) || currentReserve.Eq(referenceReserve) {
		left, err := addChecked(&currentAfter, currentReserve)
		if err != nil {
			return zero, err
		}
		right, err := mulChecked(referenceReserve, linearStartReferenceScale)
		if err != nil {
			return zero, err
		}
		if left.Gt(&right) {
			fee, err = calculateLinearFee(input, currentReserve, referenceReserve)
		} else {
			fee, err = calculateQuadraticFee(input, currentReserve, referenceReserve)
		}
		if err != nil {
			return zero, err
		}
	} else if currentAfter.Gt(referenceReserve) {
		pastBy, err := subChecked(&currentAfter, referenceReserve)
		if err != nil {
			return zero, err
		}
		twiceRef, err := mulChecked(referenceReserve, two)
		if err != nil {
			return zero, err
		}
		if pastBy.Gt(&twiceRef) {
			fee, err = calculateLinearFee(&pastBy, referenceReserve, referenceReserve)
		} else {
			fee, err = calculateQuadraticFee(&pastBy, referenceReserve, referenceReserve)
		}
		if err != nil {
			return zero, err
		}
		weighted, err := mulChecked(&fee, &pastBy)
		if err != nil {
			return zero, err
		}
		fee, err = divChecked(&weighted, input)
		if err != nil {
			return zero, err
		}
	}
	if fee.Lt(minFeeQ64) {
		fee.Set(minFeeQ64)
	}
	return fee, nil
}

func calculateQuadraticFee(input, currentReserve, referenceReserve *uint256.Int) (uint256.Int, error) {
	var result uint256.Int
	delta, err := subChecked(currentReserve, referenceReserve)
	if err != nil {
		return result, err
	}
	delta, err = mulChecked(&delta, two)
	if err != nil {
		return result, err
	}
	base, err := addChecked(input, &delta)
	if err != nil {
		return result, err
	}
	numerator, err := mulChecked(feeScaleQ64, &base)
	if err != nil {
		return result, err
	}
	return divChecked(&numerator, referenceReserve)
}

func calculateLinearFee(input, currentReserve, referenceReserve *uint256.Int) (uint256.Int, error) {
	var result uint256.Int
	delta, err := subChecked(currentReserve, referenceReserve)
	if err != nil {
		return result, err
	}
	delta, err = mulChecked(&delta, two)
	if err != nil {
		return result, err
	}
	denomBase, err := addChecked(input, &delta)
	if err != nil {
		return result, err
	}
	denominator, err := mulChecked(twenty, &denomBase)
	if err != nil {
		return result, err
	}
	numerator, err := mulChecked(referenceReserve, maxQuadraticFeeQ64)
	if err != nil {
		return result, err
	}
	quotient, err := divChecked(&numerator, &denominator)
	if err != nil {
		return result, err
	}
	inner, err := subChecked(twoQ64, &quotient)
	if err != nil {
		return result, err
	}
	fee, err := mulChecked(fourThousand, &inner)
	if err != nil {
		return result, err
	}
	return fee, nil
}

func actualToAdjusted(actual, missing *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	if missing.Gt(actual) || missing.Eq(actual) {
		return zero, errMissingGteActual
	}
	left, err := mulChecked(missing, bufferNumerator)
	if err != nil {
		return zero, err
	}
	right, err := mulChecked(actual, buffer)
	if err != nil {
		return zero, err
	}
	if left.Lt(&right) {
		return mulChecked(actual, one)
	}
	depleted, err := subChecked(actual, missing)
	if err != nil {
		return zero, err
	}
	return mulChecked(&depleted, bufferNumerator)
}

func adjustedToActual(adjusted, missing *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	divisor, _ := subChecked(bufferNumerator, buffer)
	ceil, err := ceilDiv(adjusted, &divisor)
	if err != nil {
		return zero, err
	}
	left, err := mulChecked(missing, bufferNumerator)
	if err != nil {
		return zero, err
	}
	right, err := mulChecked(&ceil, buffer)
	if err != nil {
		return zero, err
	}
	if left.Lt(&right) {
		return ceil, nil
	}
	base, err := ceilDiv(adjusted, bufferNumerator)
	if err != nil {
		return zero, err
	}
	return addChecked(&base, missing)
}

func calculateBalanceAfterFees(amountIn, balance, reserve, referenceReserve, missing *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	fee, err := calculateSwapFeeBipsQ64(amountIn, reserve, referenceReserve)
	if err != nil {
		return zero, err
	}
	left, err := mulChecked(balance, buffer)
	if err != nil {
		return zero, err
	}
	right, err := mulChecked(missing, bufferNumerator)
	if err != nil {
		return zero, err
	}
	base := *balance
	depleted := left.Lt(&right)
	if depleted {
		base, err = subChecked(&base, missing)
		if err != nil {
			return zero, err
		}
	}
	base, err = mulChecked(&base, bipsQ64)
	if err != nil {
		return zero, err
	}
	feeAmount, err := mulChecked(amountIn, &fee)
	if err != nil {
		return zero, err
	}
	base, err = subChecked(&base, &feeAmount)
	if err != nil {
		return zero, err
	}
	if depleted {
		base, err = mulChecked(&base, bufferNumerator)
		if err != nil {
			return zero, err
		}
	}
	return divChecked(&base, bipsQ64)
}

func quoteExactIn(amountIn, reserveIn, referenceReserveIn, reserveOut, missingIn, missingOut *uint256.Int) (uint256.Int, error) {
	var zero uint256.Int
	balance, err := addChecked(reserveIn, amountIn)
	if err != nil {
		return zero, err
	}
	adjustedInput, err := calculateBalanceAfterFees(amountIn, &balance, reserveIn, referenceReserveIn, missingIn)
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
	firstAdjusted, err := ceilDiv(&k, &adjustedInput)
	if err != nil {
		return zero, err
	}
	newReserveOut, err := adjustedToActual(&firstAdjusted, missingOut)
	if err != nil {
		return zero, err
	}
	return subChecked(reserveOut, &newReserveOut)
}
