package math

import (
	"errors"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var (
	ErrAddOverflow  = errors.New("ADD_OVERFLOW")
	ErrSubOverflow  = errors.New("SUB_OVERFLOW")
	ErrZeroDivision = errors.New("ZERO_DIVISION")
	ErrDivInternal  = errors.New("DIV_INTERNAL")
	ErrMulOverflow  = errors.New("MUL_OVERFLOW")

	ONE_E18                = uint256.NewInt(1e18) // 18 decimal places
	TWO_E18                = new(uint256.Int).Mul(ONE_E18, TWO)
	FOUR_E18               = new(uint256.Int).Mul(TWO_E18, TWO)
	MAX_POW_RELATIVE_ERROR = uint256.NewInt(10000) // 10^(-14)
)

var FixPoint *fixPoint

type fixPoint struct{}

func init() {
	FixPoint = &fixPoint{}
}

func (f *fixPoint) MulDivUp(a, b, c *uint256.Int) (*uint256.Int, error) {
	return v3Utils.MulDivRoundingUp(a, b, c)
}

func (f *fixPoint) MulUp(a, b *uint256.Int) (*uint256.Int, error) {
	return v3Utils.MulDivRoundingUp(a, b, ONE_E18)
}

func (f *fixPoint) MulDown(a, b *uint256.Int) (*uint256.Int, error) {
	res, overflow := new(uint256.Int).MulDivOverflow(a, b, ONE_E18)
	if overflow {
		return nil, ErrMulOverflow
	}

	return res, nil
}

func (f *fixPoint) DivUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	return v3Utils.MulDivRoundingUp(a, ONE_E18, b)
}

func (f *fixPoint) DivDown(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	res, overflow := new(uint256.Int).MulDivOverflow(a, ONE_E18, b)
	if overflow {
		return nil, ErrMulOverflow
	}

	return res, nil
}

func (f *fixPoint) DivRawUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	// result = a == 0 ? 0 : 1 + (a - 1) / b
	if a.IsZero() {
		return ZERO, nil
	}

	delta := new(uint256.Int).Sub(a, ONE)
	delta.Div(delta, b)
	delta.Add(ONE, delta)

	return delta, nil
}

func (f *fixPoint) PowUp(x, y *uint256.Int) (*uint256.Int, error) {
	if y.Eq(ONE_E18) {
		return x, nil
	}

	if y.Eq(TWO_E18) {
		return f.MulUp(x, x)
	}

	if y.Eq(FOUR_E18) {
		square, err := f.MulUp(x, x)
		if err != nil {
			return nil, err
		}

		return f.MulUp(square, square)
	}

	raw, err := Pow(x, y)
	if err != nil {
		return nil, err
	}

	var maxError *uint256.Int
	maxError, err = f.MulUp(raw, MAX_POW_RELATIVE_ERROR)
	if err != nil {
		return nil, err
	}

	maxError, err = f.Add(maxError, ONE)
	if err != nil {
		return nil, err
	}

	return f.Add(raw, maxError)
}

func (f *fixPoint) Complement(x *uint256.Int) *uint256.Int {
	// result = (x < ONE) ? (ONE - x) : 0
	result := new(uint256.Int).Set(ZERO)
	if x.Lt(ONE_E18) {
		result.Sub(ONE_E18, x)
	}

	return result
}

func (f *fixPoint) Add(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c, overflow := new(uint256.Int).AddOverflow(a, b)
	if overflow {
		return nil, ErrAddOverflow
	}
	return c, nil
}

func (f *fixPoint) Sub(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c, overflow := new(uint256.Int).SubOverflow(a, b)
	if overflow {
		return nil, ErrSubOverflow
	}
	return c, nil
}

func (f *fixPoint) Mul(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c, overflow := new(uint256.Int).MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	}
	return c, nil
}
