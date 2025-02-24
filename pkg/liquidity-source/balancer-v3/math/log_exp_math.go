package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	iTWO      = int256.NewInt(2)
	iTHREE    = int256.NewInt(3)
	iFIVE     = int256.NewInt(5)
	iSEVEN    = int256.NewInt(7)
	iNINE     = int256.NewInt(9)
	iELEVEN   = int256.NewInt(11)
	iTHIRTEEN = int256.NewInt(13)
	iFIFTEEN  = int256.NewInt(15)
	iHUNDRED  = int256.NewInt(100)

	// 18 decimal constants
	iX0, _ = int256.FromDec("128000000000000000000")                                    // 2ˆ7
	iA0, _ = int256.FromDec("38877084059945950922200000000000000000000000000000000000") // eˆ(x0) (no decimals)
	iX1, _ = int256.FromDec("64000000000000000000")                                     // 2^6
	iA1, _ = int256.FromDec("6235149080811616882910000000")                             // eˆ(x1) (no decimals)

	// 20 decimal constants
	iX2, _  = int256.FromDec("3200000000000000000000")             // 2^5
	iA2, _  = int256.FromDec("7896296018268069516100000000000000") // eˆ(x2)
	iX3, _  = int256.FromDec("1600000000000000000000")             // 2ˆ4
	iA3, _  = int256.FromDec("888611052050787263676000000")        // eˆ(x3)
	iX4, _  = int256.FromDec("800000000000000000000")              // 2ˆ3
	iA4, _  = int256.FromDec("298095798704172827474000")           // eˆ(x4)
	iX5, _  = int256.FromDec("400000000000000000000")              // 2ˆ2
	iA5, _  = int256.FromDec("5459815003314423907810")             // eˆ(x5)
	iX6, _  = int256.FromDec("200000000000000000000")              // 2ˆ1
	iA6, _  = int256.FromDec("738905609893065022723")              // eˆ(x6)
	iX7, _  = int256.FromDec("100000000000000000000")              // 2ˆ0
	iA7, _  = int256.FromDec("271828182845904523536")              // eˆ(x7)
	iX8, _  = int256.FromDec("50000000000000000000")               // 2ˆ-1
	iA8, _  = int256.FromDec("164872127070012814685")              // eˆ(x8)
	iX9, _  = int256.FromDec("25000000000000000000")               // 2ˆ-2
	iA9, _  = int256.FromDec("128402541668774148407")              // eˆ(x9)
	iX10, _ = int256.FromDec("12500000000000000000")               // 2ˆ-3
	iA10, _ = int256.FromDec("113314845306682631683")              // eˆ(x10)
	iX11, _ = int256.FromDec("6250000000000000000")                // 2ˆ-4
	iA11, _ = int256.FromDec("106449445891785942956")              // eˆ(x11)

	iONE_E17    = int256.NewInt(1e17)                                     // 1e17
	iONE_E18    = int256.NewInt(1e18)                                     // 1e18
	iONE_E20, _ = int256.FromDec("100000000000000000000")                 // 1e20
	iONE_E36, _ = int256.FromDec("1000000000000000000000000000000000000") // 1e36

	ONE_E20, _          = uint256.FromDecimal("100000000000000000000") // 1e20
	TWO_254             = new(uint256.Int).Lsh(ONE, 254)               // 2^254
	MILD_EXPONENT_BOUND = new(uint256.Int).Div(TWO_254, ONE_E20)       // 2^254 / uint256(ONE_20)

	LN_36_LOWER_BOUND = new(int256.Int).Sub(iONE_E18, iONE_E17) // ONE_18 - 1e17
	LN_36_UPPER_BOUND = new(int256.Int).Add(iONE_E18, iONE_E17) // ONE_18 + 1e17

	MAX_NATURAL_EXPONENT = new(int256.Int).Mul(int256.NewInt(130), iONE_E18) // 130e18
	MIN_NATURAL_EXPONENT = new(int256.Int).Mul(int256.NewInt(-41), iONE_E18) // -41e18

	ErrBaseOutOfBounds     = errors.New("Base_OutOfBounds")
	ErrExponentOutOfBounds = errors.New("Exponent_OutOfBounds")
	ErrProductOutOfBounds  = errors.New("Product_OutOfBounds")
	ErrInvalidExponent     = errors.New("Invalid_Exponent")
)

func Pow(x, y *uint256.Int) (*uint256.Int, error) {
	if y.IsZero() {
		return ONE_E18, nil
	}

	if x.IsZero() {
		return ZERO, nil
	}

	xRight255 := new(uint256.Int).Rsh(x, 255)
	if !xRight255.IsZero() {
		return nil, ErrBaseOutOfBounds
	}

	if y.Cmp(MILD_EXPONENT_BOUND) >= 0 {
		return nil, ErrExponentOutOfBounds
	}

	x_int256 := i256.SafeToInt256(x)
	y_int256 := i256.SafeToInt256(y)

	var (
		logx_times_y = new(int256.Int)
		overflow     bool
	)

	if LN_36_LOWER_BOUND.Lt(x_int256) && x_int256.Lt(LN_36_UPPER_BOUND) {
		ln_36_x, err := Ln36(x_int256)
		if err != nil {
			return nil, err
		}

		// logx_times_y = ((ln_36_x / ONE_18) * y_int256 + ((ln_36_x % ONE_18) * y_int256) / ONE_18)
		quotient := new(int256.Int).Quo(ln_36_x, iONE_E18)
		remainder := new(int256.Int).Rem(ln_36_x, iONE_E18)

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
		term2 = term2.Quo(term2, iONE_E18)

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

	logx_times_y = logx_times_y.Quo(logx_times_y, iONE_E18)

	if !(MIN_NATURAL_EXPONENT.Cmp(logx_times_y) <= 0 && logx_times_y.Cmp(MAX_NATURAL_EXPONENT) <= 0) {
		return nil, ErrProductOutOfBounds
	}

	exp_logx_times_y, err := Exp(logx_times_y)
	if err != nil {
		return nil, err
	}

	return i256.SafeConvertToUInt256(exp_logx_times_y), nil
}

func Ln36(x *int256.Int) (*int256.Int, error) {
	x18, overflow := new(int256.Int).MulOverflow(x, iONE_E18)
	if overflow {
		return nil, ErrMulOverflow
	}

	// z = (x - ONE_36) * ONE_36 / (x + ONE_36)
	numerator := new(int256.Int).Sub(x18, iONE_E36)
	numerator, overflow = numerator.MulOverflow(numerator, iONE_E36)
	if overflow {
		return nil, ErrMulOverflow
	}

	denominator, overflow := new(int256.Int).AddOverflow(x18, iONE_E36)
	if overflow {
		return nil, ErrMulOverflow
	}

	z := new(int256.Int).Quo(numerator, denominator)

	// z_squared = (z * z) / ONE_36
	zSquared := new(int256.Int).Pow(z, 2)
	zSquared.Quo(zSquared, iONE_E36)

	num := new(int256.Int).Set(z)
	seriesSum := new(int256.Int).Set(z)

	temp := new(int256.Int)

	// Helper function for term calculation
	calculateTerm := func(divisor *int256.Int) error {
		num, overflow = num.MulOverflow(num, zSquared)
		if overflow {
			return ErrMulOverflow
		}
		num.Quo(num, iONE_E36)

		temp.Set(num)
		temp.Quo(temp, divisor)

		seriesSum, overflow = seriesSum.AddOverflow(seriesSum, temp)
		if overflow {
			return ErrAddOverflow
		}

		return nil
	}

	// Calculate all terms
	for _, divisor := range []*int256.Int{iTHREE, iFIVE, iSEVEN, iNINE, iELEVEN, iTHIRTEEN, iFIFTEEN} {
		if err := calculateTerm(divisor); err != nil {
			return nil, err
		}
	}

	result, overflow := seriesSum.MulOverflow(seriesSum, iTWO)
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

	if a.Lt(iONE_E18) {
		numerator, overflow = numerator.MulOverflow(iONE_E18, iONE_E18)
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

	temp, overflow = temp.MulOverflow(iA0, iONE_E18)
	if overflow {
		return nil, ErrMulOverflow
	}
	if a.Cmp(temp) >= 0 {
		a.Quo(a, iA0)
		sum.Add(sum, iX0)
	}

	temp, overflow = temp.MulOverflow(iA1, iONE_E18)
	if overflow {
		return nil, ErrMulOverflow
	}
	if a.Cmp(temp) >= 0 {
		a.Quo(a, iA1)
		sum.Add(sum, iX1)
	}

	sum, overflow = sum.MulOverflow(sum, iHUNDRED)
	if overflow {
		return nil, ErrMulOverflow
	}
	a, overflow = a.MulOverflow(a, iHUNDRED)
	if overflow {
		return nil, ErrMulOverflow
	}

	tempMul := new(int256.Int)
	checkAndAdd := func(an, xn *int256.Int) error {
		if a.Cmp(an) >= 0 {
			tempMul, overflow = tempMul.MulOverflow(a, iONE_E20)
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
	aMinus, overflow = aMinus.SubOverflow(a, iONE_E20)
	if overflow {
		return nil, ErrSubOverflow
	}

	aPlus := new(int256.Int)
	aPlus, overflow = aPlus.AddOverflow(a, iONE_E20)
	if overflow {
		return nil, ErrAddOverflow
	}

	numerator, overflow = numerator.MulOverflow(aMinus, iONE_E20)
	if overflow {
		return nil, ErrMulOverflow
	}

	z := new(int256.Int).Quo(numerator, aPlus)
	zSquared, overflow := new(int256.Int).MulOverflow(z, z)
	if overflow {
		return nil, ErrMulOverflow
	}
	zSquared.Quo(zSquared, iONE_E20)

	num := new(int256.Int).Set(z)
	seriesSum := new(int256.Int).Set(z)
	temp2 := new(int256.Int)

	// Helper function for term calculation
	calculateTerm := func(divisor *int256.Int) error {
		num, overflow = num.MulOverflow(num, zSquared)
		if overflow {
			return ErrMulOverflow
		}
		num.Quo(num, iONE_E20)

		temp2.Set(num)
		temp2.Quo(temp2, divisor)

		seriesSum, overflow = seriesSum.AddOverflow(seriesSum, temp2)
		if overflow {
			return ErrAddOverflow
		}
		return nil
	}

	for _, divisor := range []*int256.Int{iTHREE, iFIVE, iSEVEN, iNINE, iELEVEN} {
		if err := calculateTerm(divisor); err != nil {
			return nil, err
		}
	}

	seriesSum, overflow = seriesSum.MulOverflow(seriesSum, iTWO)
	if overflow {
		return nil, ErrMulOverflow
	}

	result := new(int256.Int).Add(sum, seriesSum)
	result.Quo(result, iHUNDRED)

	if negativeExponent {
		result.Neg(result)
	}

	return result, nil
}

func Exp(x *int256.Int) (*int256.Int, error) {
	if x.Lt(MIN_NATURAL_EXPONENT) || x.Gt(MAX_NATURAL_EXPONENT) {
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
	x, overflow = x.MulOverflow(x, iHUNDRED)
	if overflow {
		return nil, ErrMulOverflow
	}

	product := new(int256.Int).Set(iONE_E20)

	// Helper function for checking and multiplying
	checkAndMultiply := func(xn, an *int256.Int) error {
		if x.Cmp(xn) >= 0 {
			x.Sub(x, xn)
			temp, overflow = temp.MulOverflow(product, an)
			if overflow {
				return ErrMulOverflow
			}
			product.Quo(temp, iONE_E20)
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

	seriesSum := new(int256.Int).Set(iONE_E20)
	term := new(int256.Int).Set(x)
	termTemp := new(int256.Int)

	// Helper function for term calculation
	calculateTerm := func(n int64) error {
		// term = ((term * x) / ONE_20) / n
		termTemp, overflow = termTemp.MulOverflow(term, x)
		if overflow {
			return ErrMulOverflow
		}
		term.Quo(termTemp, iONE_E20)
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
	temp.Quo(temp, iONE_E20)

	result, overflow := temp.MulOverflow(temp, firstAN)
	if overflow {
		return nil, ErrMulOverflow
	}
	result.Quo(result, iHUNDRED)

	if negativeExponent {
		numerator, overflow := new(int256.Int).MulOverflow(iONE_E18, iONE_E18)
		if overflow {
			return nil, ErrMulOverflow
		}
		result.Quo(numerator, result)
	}

	return result, nil
}
