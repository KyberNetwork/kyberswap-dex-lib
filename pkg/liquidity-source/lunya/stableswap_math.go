package lunya

import (
	"github.com/holiman/uint256"
)

// A port of StableSwapMath. The Solidity runs almost entirely in `unchecked` blocks, which uint256's
// wrapping arithmetic reproduces as it is; FullMath.mulDiv's reverts become errors.

var (
	one             = uint256.NewInt(1)
	three           = uint256.NewInt(3)
	aPrecisionU     = uint256.NewInt(aPrecision)
	feeDenominatorU = uint256.NewInt(feeDenominator)

	two64      = new(uint256.Int).Lsh(one, 64)
	q96        = new(uint256.Int).Lsh(one, 96)
	two128     = new(uint256.Int).Lsh(one, 128)
	q192       = new(uint256.Int).Lsh(one, 192)
	maxUint128 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(one, 128), 1)
	maxUint160 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(one, 160), 1)
	maxUint256 = new(uint256.Int).SetAllOne()
	maxInt256  = new(uint256.Int).Rsh(maxUint256, 1)
)

// mulDiv is FullMath.mulDiv: floor(x * y / d) over the 512-bit product.
func mulDiv(z, x, y, d *uint256.Int) error {
	if d.IsZero() {
		return ErrArithmetic
	}
	if _, overflow := z.MulDivOverflow(x, y, d); overflow {
		return ErrArithmetic
	}
	return nil
}

// mulDivRoundingUp is FullMath.mulDivRoundingUp.
func mulDivRoundingUp(z, x, y, d *uint256.Int) error {
	var remainder uint256.Int
	remainder.MulMod(x, y, d) // before z is written: z may alias x or y
	if err := mulDiv(z, x, y, d); err != nil {
		return err
	}
	if !remainder.IsZero() {
		if z.Eq(maxUint256) {
			return ErrArithmetic
		}
		z.AddUint64(z, 1)
	}
	return nil
}

func absDiff(z, a, b *uint256.Int) {
	if a.Gt(b) {
		z.Sub(a, b)
	} else {
		z.Sub(b, a)
	}
}

// subtractFee and addFee are StableSwapMath's: the net that reaches the curve from a gross input, and
// back.
func subtractFee(z, amountInGross *uint256.Int, fee uint64) error {
	if fee >= feeDenominator {
		return ErrArithmetic
	}
	var complement uint256.Int
	complement.SetUint64(feeDenominator - fee)
	return mulDiv(z, amountInGross, &complement, feeDenominatorU)
}

func addFee(z, amountInNet *uint256.Int, fee uint64) error {
	if fee >= feeDenominator {
		return ErrArithmetic
	}
	var complement uint256.Int
	complement.SetUint64(feeDenominator - fee)
	return mulDivRoundingUp(z, amountInNet, feeDenominatorU, &complement)
}

// getD is the StableSwap invariant by Newton's method. It stops within one unit, or where the step has
// stopped shrinking, which is the limit cycle integer arithmetic settles into at large magnitudes.
func getD(x, y, ann *uint256.Int) (uint256.Int, error) {
	var sum, d uint256.Int
	sum.Add(x, y)
	if sum.IsZero() {
		return d, nil
	}
	d.Set(&sum)

	var x2, y2, annLessPrecision uint256.Int
	x2.Lsh(x, 1)
	y2.Lsh(y, 1)
	annLessPrecision.Sub(ann, aPrecisionU)

	prevDelta := *maxUint256
	var d2Over2x, dP, dPrev, numerator, denominator, term, delta uint256.Int
	for range maxIter {
		// D_P = D^3 / (4xy), as D^2/(2x) * D/(2y)
		if err := mulDiv(&d2Over2x, &d, &d, &x2); err != nil {
			return d, err
		}
		if err := mulDiv(&dP, &d2Over2x, &d, &y2); err != nil {
			return d, err
		}
		dPrev.Set(&d)

		// ((ann*S + 2*D_P*A_PRECISION) * D) / ((ann - A_PRECISION)*D + 3*D_P*A_PRECISION)
		numerator.Mul(ann, &sum)
		term.Lsh(&dP, 1).Mul(&term, aPrecisionU)
		numerator.Add(&numerator, &term)
		denominator.Mul(&annLessPrecision, &d)
		term.Mul(&dP, three).Mul(&term, aPrecisionU)
		denominator.Add(&denominator, &term)
		if err := mulDiv(&d, &numerator, &d, &denominator); err != nil {
			return d, err
		}

		absDiff(&delta, &d, &dPrev)
		if !delta.Gt(one) || !delta.Lt(&prevDelta) {
			return d, nil
		}
		prevDelta.Set(&delta)
	}
	return d, nil
}

// getOtherReserve solves the curve for one reserve given the other and D, by Newton's method from seed.
func getOtherReserve(known, d, ann, seed *uint256.Int) (uint256.Int, error) {
	var c, b, other, tmp uint256.Int

	// c = D^3 / (4 * known * Ann), without forming D^3
	tmp.Lsh(known, 1)
	if err := mulDiv(&c, d, d, &tmp); err != nil {
		return other, err
	}
	var dScaled, ann2 uint256.Int
	dScaled.Mul(d, aPrecisionU)
	ann2.Lsh(ann, 1)
	if err := mulDiv(&c, &c, &dScaled, &ann2); err != nil {
		return other, err
	}
	// b = known + D/Ann
	if err := mulDiv(&tmp, d, aPrecisionU, ann); err != nil {
		return other, err
	}
	b.Add(known, &tmp)

	other.Set(seed)
	prevDelta := *maxUint256
	var prev, numerator, denominator, delta uint256.Int
	for range maxIter {
		prev.Set(&other)
		// v' = (v^2 + c) / (2v + b - D); the square is checked, as mulDiv(v, v, 1) is
		if _, overflow := numerator.MulOverflow(&other, &other); overflow {
			return other, ErrArithmetic
		}
		numerator.Add(&numerator, &c)
		denominator.Lsh(&other, 1).Add(&denominator, &b)
		if !denominator.Gt(d) {
			return other, ErrArithmetic
		}
		denominator.Sub(&denominator, d)
		other.Div(&numerator, &denominator)

		absDiff(&delta, &other, &prev)
		if !delta.Gt(one) || !delta.Lt(&prevDelta) {
			return other, nil
		}
		prevDelta.Set(&delta)
	}
	return other, nil
}

// ratioQ192 is floor(num * 2^192 / den), saturating where it would not fit.
func ratioQ192(z, num, den *uint256.Int) error {
	var quotient uint256.Int
	if den.IsZero() || !quotient.Div(num, den).Lt(two64) {
		z.Set(maxUint256)
		return nil
	}
	return mulDiv(z, num, q192, den)
}

// ratioQ128 is floor(num * 2^128 / den), saturating where it would not fit.
func ratioQ128(z, num, den *uint256.Int) error {
	var quotient uint256.Int
	if den.IsZero() || !quotient.Div(num, den).Lt(two128) {
		z.Set(maxUint256)
		return nil
	}
	return mulDiv(z, num, two128, den)
}

// sqrtPriceFromReserves is the Q64.96 sqrt price y/x. Its square root is ported step for step rather
// than replaced by a floor square root: the last correction can land one below the floor.
func sqrtPriceFromReserves(x, y *uint256.Int) (uint256.Int, error) {
	var result uint256.Int
	if x.IsZero() {
		return *maxUint160, nil
	}

	var ratio uint256.Int
	if err := ratioQ192(&ratio, y, x); err != nil {
		return result, err
	}
	if ratio.IsZero() {
		return result, nil
	}

	v := ratio
	result.SetOne()
	for _, bits := range []uint{128, 64, 32, 16, 8, 4} {
		if v.BitLen() > int(bits) {
			v.Rsh(&v, bits)
			result.Lsh(&result, bits/2)
		}
	}
	if v.BitLen() > 2 {
		result.Lsh(&result, 1)
	}

	var quotient uint256.Int
	for range 7 {
		quotient.Div(&ratio, &result)
		result.Add(&result, &quotient).Rsh(&result, 1)
	}
	if quotient.Div(&ratio, &result); quotient.Lt(&result) {
		result.Set(&quotient)
	}

	if result.Gt(maxUint160) {
		return result, ErrPriceOutOfRange
	}
	return result, nil
}

// bisectToTargetPrice finds the reserve pair on the curve at targetPrice. Price is y/x: with searchX it
// bisects x and returns (x, y), otherwise it bisects y and returns (y, x).
func bisectToTargetPrice(lo, hi, d, ann, targetPrice *uint256.Int, searchX bool) (v, other uint256.Int,
	err error) {
	var targetRatio uint256.Int
	if err = mulDiv(&targetRatio, targetPrice, targetPrice, two64); err != nil {
		return
	}

	low, high := *lo, *hi
	om := *d
	var mid, currentRatio, span uint256.Int
	for range bisectRounds {
		mid.Add(&low, &high).Rsh(&mid, 1)
		if om, err = getOtherReserve(&mid, d, ann, &om); err != nil {
			return
		}
		if searchX {
			err = ratioQ128(&currentRatio, &om, &mid)
		} else {
			err = ratioQ128(&currentRatio, &mid, &om)
		}
		if err != nil {
			return
		}
		if currentRatio.Eq(&targetRatio) {
			return mid, om, nil
		}
		if searchX && currentRatio.Gt(&targetRatio) || !searchX && currentRatio.Lt(&targetRatio) {
			low = mid
		} else {
			high = mid
		}
		if !span.Sub(&high, &low).Gt(one) {
			break
		}
	}

	v = high
	other, err = getOtherReserve(&v, d, ann, &om)
	return
}

// movePriceTowardsTarget is the swap on the curve, in 18-decimal units. amountAvailable is an int256 in
// two's complement: positive for exact input, negative for exact output. It returns the price after the
// swap, the net input, the output and the fee on the input leg.
func movePriceTowardsTarget(zeroToOne bool, targetPrice, x0, y0, amountAvailable *uint256.Int, fee uint64,
	ann *uint256.Int) (resultPrice, input, output, feeAmount uint256.Int, err error) {
	if x0.IsZero() || y0.IsZero() {
		err = ErrArithmetic
		return
	}
	if targetPrice.IsZero() {
		err = ErrPriceOutOfRange
		return
	}
	if amountAvailable.IsZero() {
		resultPrice, err = sqrtPriceFromReserves(x0, y0)
		return
	}

	d, err := getD(x0, y0, ann)
	if err != nil {
		return
	}

	var x1, y1, two uint256.Int
	two.SetUint64(2)

	if amountAvailable.Sign() >= 0 {
		// exact input
		var net uint256.Int
		if err = subtractFee(&net, amountAvailable, fee); err != nil {
			return
		}
		if net.IsZero() {
			resultPrice, err = sqrtPriceFromReserves(x0, y0)
			return
		}

		if zeroToOne {
			x1.Add(x0, &net)
			if y1, err = getOtherReserve(&x1, &d, ann, &d); err != nil {
				return
			}
			if resultPrice, err = sqrtPriceFromReserves(&x1, &y1); err != nil {
				return
			}
			// 0 -> 1 must not push the price below the target
			if resultPrice.Lt(targetPrice) {
				if x1, y1, err = bisectToTargetPrice(x0, &x1, &d, ann, targetPrice, true); err != nil {
					return
				}
				resultPrice.Set(targetPrice)
				net.Sub(&x1, x0)
			}
			input = net
			// the payout is rounded down by two units, one per leg of a round trip
			if bound := new(uint256.Int).Add(&y1, &two); y0.Gt(bound) {
				output.Sub(y0, bound)
			}
		} else {
			y1.Add(y0, &net)
			if x1, err = getOtherReserve(&y1, &d, ann, &d); err != nil {
				return
			}
			if resultPrice, err = sqrtPriceFromReserves(&x1, &y1); err != nil {
				return
			}
			// 1 -> 0 must not push the price above the target
			if resultPrice.Gt(targetPrice) {
				if y1, x1, err = bisectToTargetPrice(y0, &y1, &d, ann, targetPrice, false); err != nil {
					return
				}
				resultPrice.Set(targetPrice)
				net.Sub(&y1, y0)
			}
			input = net
			if bound := new(uint256.Int).Add(&x1, &two); x0.Gt(bound) {
				output.Sub(x0, bound)
			}
		}

		if err = addFee(&feeAmount, &input, fee); err != nil {
			return
		}
		feeAmount.Sub(&feeAmount, &input)
		return
	}

	// exact output
	output.Neg(amountAvailable)
	var feeComplement uint256.Int
	feeComplement.SetUint64(feeDenominator - fee)

	if zeroToOne {
		// token1 out
		if !output.Lt(y0) {
			err = ErrInsufficientReserve
			return
		}
		y1.Sub(y0, &output)
		if x1, err = getOtherReserve(&y1, &d, ann, &d); err != nil {
			return
		}
		if resultPrice, err = sqrtPriceFromReserves(&x1, &y1); err != nil {
			return
		}
		if resultPrice.Lt(targetPrice) {
			if x1, y1, err = bisectToTargetPrice(x0, &x1, &d, ann, targetPrice, true); err != nil {
				return
			}
			resultPrice.Set(targetPrice)
			output.Sub(y0, &y1)
		}
		// anything handed out costs at least one unit
		if x1.Gt(x0) {
			input.Sub(&x1, x0).AddUint64(&input, 1)
		} else {
			input.SetOne()
		}
	} else {
		// token0 out
		if !output.Lt(x0) {
			err = ErrInsufficientReserve
			return
		}
		x1.Sub(x0, &output)
		if y1, err = getOtherReserve(&x1, &d, ann, &d); err != nil {
			return
		}
		if resultPrice, err = sqrtPriceFromReserves(&x1, &y1); err != nil {
			return
		}
		if resultPrice.Gt(targetPrice) {
			if y1, x1, err = bisectToTargetPrice(y0, &y1, &d, ann, targetPrice, false); err != nil {
				return
			}
			resultPrice.Set(targetPrice)
			output.Sub(x0, &x1)
		}
		if y1.Gt(y0) {
			input.Sub(&y1, y0).AddUint64(&input, 1)
		} else {
			input.SetOne()
		}
	}

	var feeU uint256.Int
	feeU.SetUint64(fee)
	err = mulDivRoundingUp(&feeAmount, &input, &feeU, &feeComplement)
	return
}
