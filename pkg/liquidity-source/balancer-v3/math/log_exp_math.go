package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	ONE_E20, _ = uint256.FromDecimal("100000000000000000000") // 1e20

	iONE_E17 = int256.NewInt(1e17)
	iONE_E18 = int256.NewInt(1e18)

	TWO_254             = new(uint256.Int).Lsh(ONE, 254)
	MILD_EXPONENT_BOUND = new(uint256.Int).Div(TWO_254, ONE_E20)

	LN_36_LOWER_BOUND = new(int256.Int).Sub(iONE_E18, iONE_E17)
	LN_36_UPPER_BOUND = new(int256.Int).Add(iONE_E18, iONE_E17)

	MAX_NATURAL_EXPONENT = new(int256.Int).Mul(int256.NewInt(130), iONE_E18) // 130e18
	MIN_NATURAL_EXPONENT = new(int256.Int).Mul(int256.NewInt(-41), iONE_E18) // -41e18

	ErrBaseOutOfBounds     = errors.New("Base_OutOfBounds")
	ErrExponentOutOfBounds = errors.New("Exponent_OutOfBounds")
	ErrProductOutOfBounds  = errors.New("Product_OutOfBounds")
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
		logx_times_y *int256.Int
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
	return nil, nil
}

func Ln(x *int256.Int) (*int256.Int, error) {
	return nil, nil
}

func Exp(x *int256.Int) (*int256.Int, error) {
	return nil, nil
}
