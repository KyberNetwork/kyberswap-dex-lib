package math

import (
	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func Pow(x, y *uint256.Int) (*uint256.Int, error) {
	if y.IsZero() {
		return U1e18, nil
	}

	if x.IsZero() {
		return U0, nil
	}

	xRight255 := new(uint256.Int).Rsh(x, 255)
	if !xRight255.IsZero() {
		return nil, ErrBaseOutOfBounds
	}

	if y.Cmp(UMildExponentBound) >= 0 {
		return nil, ErrExponentOutOfBounds
	}

	x_int256 := i256.SafeToInt256(x)
	y_int256 := i256.SafeToInt256(y)

	var (
		logx_times_y = new(int256.Int)
		overflow     bool
	)

	if ILn36LowerBound.Lt(x_int256) && x_int256.Lt(ILn36UpperBound) {
		ln_36_x, err := Ln36(x_int256)
		if err != nil {
			return nil, err
		}

		// logx_times_y = ((ln_36_x / ONE_18) * y_int256 + ((ln_36_x % ONE_18) * y_int256) / ONE_18)
		quotient := new(int256.Int).Quo(ln_36_x, i1e18)
		remainder := new(int256.Int).Rem(ln_36_x, i1e18)

		// (ln36X / ONE_18) * y_int256
		term1, overflow := new(int256.Int).MulOverflow(quotient, y_int256)
		if overflow {
			return nil, ErrMulOverflow
		}

		// ((ln36X % ONE_18) * y_int256) / ONE_18
		term2, overflow := new(int256.Int).MulOverflow(remainder, y_int256)
		if overflow {
			return nil, ErrMulOverflow
		}
		term2 = term2.Quo(term2, i1e18)

		logx_times_y, overflow = logx_times_y.AddOverflow(term1, term2)
		if overflow {
			return nil, ErrAddOverflow
		}
	} else {
		ln_x, err := Ln(x_int256)
		if err != nil {
			return nil, err
		}

		logx_times_y, overflow = logx_times_y.MulOverflow(ln_x, y_int256)
		if overflow {
			return nil, ErrMulOverflow
		}
	}

	logx_times_y = logx_times_y.Quo(logx_times_y, i1e18)

	if !(IMinNaturalExponent.Cmp(logx_times_y) <= 0 && logx_times_y.Cmp(IMaxNaturalExponent) <= 0) {
		return nil, ErrProductOutOfBounds
	}

	exp_logx_times_y, err := Exp(logx_times_y)
	if err != nil {
		return nil, err
	}

	return i256.SafeConvertToUInt256(exp_logx_times_y), nil
}

func Ln36(x *int256.Int) (*int256.Int, error) {
	x18, overflow := new(int256.Int).MulOverflow(x, i1e18)
	if overflow {
		return nil, ErrMulOverflow
	}

	// z = (x - ONE_36) * ONE_36 / (x + ONE_36)
	numerator := new(int256.Int).Sub(x18, i1e36)
	numerator, overflow = numerator.MulOverflow(numerator, i1e36)
	if overflow {
		return nil, ErrMulOverflow
	}

	denominator, overflow := new(int256.Int).AddOverflow(x18, i1e36)
	if overflow {
		return nil, ErrMulOverflow
	}

	z := new(int256.Int).Quo(numerator, denominator)

	// z_squared = (z * z) / ONE_36
	zSquared := new(int256.Int).Mul(z, z)
	zSquared.Quo(zSquared, i1e36)

	num := new(int256.Int).Set(z)
	seriesSum := new(int256.Int).Set(z)

	temp := new(int256.Int)

	// Helper function for term calculation
	calculateTerm := func(divisor *int256.Int) error {
		num, overflow = num.MulOverflow(num, zSquared)
		if overflow {
			return ErrMulOverflow
		}
		num.Quo(num, i1e36)

		temp.Set(num)
		temp.Quo(temp, divisor)

		seriesSum, overflow = seriesSum.AddOverflow(seriesSum, temp)
		if overflow {
			return ErrAddOverflow
		}

		return nil
	}

	// Calculate all terms
	for _, divisor := range []*int256.Int{i3, i5, i7, i9, i11, i13, i15} {
		if err := calculateTerm(divisor); err != nil {
			return nil, err
		}
	}

	result, overflow := seriesSum.MulOverflow(seriesSum, I2)
	if overflow {
		return nil, ErrMulOverflow
	}

	return result, nil
}

func Ln(a *int256.Int) (*int256.Int, error) {
	var (
		numerator        = new(int256.Int)
		negativeExponent = false
		overflow         bool
	)

	if a.Lt(i1e18) {
		numerator, overflow = numerator.MulOverflow(i1e18, i1e18)
		if overflow {
			return nil, ErrMulOverflow
		}

		a.Quo(numerator, a)
		negativeExponent = true
	}

	var (
		sum  = new(int256.Int)
		temp = new(int256.Int)
	)

	temp, overflow = temp.MulOverflow(iA0, i1e18)
	if overflow {
		return nil, ErrMulOverflow
	}
	if a.Cmp(temp) >= 0 {
		a.Quo(a, iA0)
		sum.Add(sum, iX0)
	}

	temp, overflow = temp.MulOverflow(iA1, i1e18)
	if overflow {
		return nil, ErrMulOverflow
	}
	if a.Cmp(temp) >= 0 {
		a.Quo(a, iA1)
		sum.Add(sum, iX1)
	}

	sum, overflow = sum.MulOverflow(sum, i100)
	if overflow {
		return nil, ErrMulOverflow
	}
	a, overflow = a.MulOverflow(a, i100)
	if overflow {
		return nil, ErrMulOverflow
	}

	tempMul := new(int256.Int)
	checkAndAdd := func(an, xn *int256.Int) error {
		if a.Cmp(an) >= 0 {
			tempMul, overflow = tempMul.MulOverflow(a, i1e20)
			if overflow {
				return ErrMulOverflow
			}
			a.Quo(tempMul, an)
			sum.Add(sum, xn)
		}
		return nil
	}

	for _, term := range [][2]*int256.Int{
		{iA2, iX2}, {iA3, iX3}, {iA4, iX4}, {iA5, iX5},
		{iA6, iX6}, {iA7, iX7}, {iA8, iX8}, {iA9, iX9},
		{iA10, iX10}, {iA11, iX11},
	} {
		if err := checkAndAdd(term[0], term[1]); err != nil {
			return nil, err
		}
	}

	aMinus := new(int256.Int)
	aMinus, overflow = aMinus.SubOverflow(a, i1e20)
	if overflow {
		return nil, ErrSubOverflow
	}

	aPlus := new(int256.Int)
	aPlus, overflow = aPlus.AddOverflow(a, i1e20)
	if overflow {
		return nil, ErrAddOverflow
	}

	numerator, overflow = numerator.MulOverflow(aMinus, i1e20)
	if overflow {
		return nil, ErrMulOverflow
	}

	z := new(int256.Int).Quo(numerator, aPlus)
	zSquared, overflow := new(int256.Int).MulOverflow(z, z)
	if overflow {
		return nil, ErrMulOverflow
	}
	zSquared.Quo(zSquared, i1e20)

	num := new(int256.Int).Set(z)
	seriesSum := new(int256.Int).Set(z)
	temp2 := new(int256.Int)

	// Helper function for term calculation
	calculateTerm := func(divisor *int256.Int) error {
		num, overflow = num.MulOverflow(num, zSquared)
		if overflow {
			return ErrMulOverflow
		}
		num.Quo(num, i1e20)

		temp2.Set(num)
		temp2.Quo(temp2, divisor)

		seriesSum, overflow = seriesSum.AddOverflow(seriesSum, temp2)
		if overflow {
			return ErrAddOverflow
		}
		return nil
	}

	for _, divisor := range []*int256.Int{i3, i5, i7, i9, i11} {
		if err := calculateTerm(divisor); err != nil {
			return nil, err
		}
	}

	seriesSum, overflow = seriesSum.MulOverflow(seriesSum, I2)
	if overflow {
		return nil, ErrMulOverflow
	}

	result := new(int256.Int).Add(sum, seriesSum)
	result.Quo(result, i100)

	if negativeExponent {
		result.Neg(result)
	}

	return result, nil
}

func Exp(x *int256.Int) (*int256.Int, error) {
	if x.Lt(IMinNaturalExponent) || x.Gt(IMaxNaturalExponent) {
		return nil, ErrExponentOutOfBounds
	}

	negativeExponent := false
	if x.Sign() < 0 {
		x.Neg(x)
		negativeExponent = true
	}

	var (
		temp    = new(int256.Int)
		firstAN = new(int256.Int).SetInt64(1)
	)

	if x.Cmp(iX0) >= 0 {
		x.Sub(x, iX0)
		firstAN.Set(iA0)
	} else if x.Cmp(iX1) >= 0 {
		x.Sub(x, iX1)
		firstAN.Set(iA1)
	}

	var overflow bool
	x, overflow = x.MulOverflow(x, i100)
	if overflow {
		return nil, ErrMulOverflow
	}

	product := new(int256.Int).Set(i1e20)

	// Helper function for checking and multiplying
	checkAndMultiply := func(xn, an *int256.Int) error {
		if x.Cmp(xn) >= 0 {
			x.Sub(x, xn)
			temp, overflow = temp.MulOverflow(product, an)
			if overflow {
				return ErrMulOverflow
			}
			product.Quo(temp, i1e20)
		}
		return nil
	}

	terms := [][2]*int256.Int{
		{iX2, iA2}, {iX3, iA3}, {iX4, iA4}, {iX5, iA5},
		{iX6, iA6}, {iX7, iA7}, {iX8, iA8}, {iX9, iA9},
	}

	for _, term := range terms {
		if err := checkAndMultiply(term[0], term[1]); err != nil {
			return nil, err
		}
	}

	seriesSum := new(int256.Int).Set(i1e20)
	term := new(int256.Int).Set(x)
	termTemp := new(int256.Int)

	// Helper function for term calculation
	calculateTerm := func(n int64) error {
		// term = ((term * x) / ONE_20) / n
		termTemp, overflow = termTemp.MulOverflow(term, x)
		if overflow {
			return ErrMulOverflow
		}
		term.Quo(termTemp, i1e20)
		term.Quo(term, int256.NewInt(n))

		seriesSum, overflow = seriesSum.AddOverflow(seriesSum, term)
		if overflow {
			return ErrAddOverflow
		}
		return nil
	}

	seriesSum, overflow = seriesSum.AddOverflow(seriesSum, x)
	if overflow {
		return nil, ErrAddOverflow
	}

	for i := int64(2); i <= 12; i++ {
		if err := calculateTerm(i); err != nil {
			return nil, err
		}
	}

	temp, overflow = temp.MulOverflow(product, seriesSum)
	if overflow {
		return nil, ErrMulOverflow
	}
	temp.Quo(temp, i1e20)

	result, overflow := temp.MulOverflow(temp, firstAN)
	if overflow {
		return nil, ErrMulOverflow
	}
	result.Quo(result, i100)

	if negativeExponent {
		numerator, overflow := new(int256.Int).MulOverflow(i1e18, i1e18)
		if overflow {
			return nil, ErrMulOverflow
		}
		result.Quo(numerator, result)
	}

	return result, nil
}
