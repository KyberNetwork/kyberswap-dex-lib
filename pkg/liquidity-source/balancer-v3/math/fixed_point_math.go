package math

import (
	"errors"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var (
	ErrAddOverflow  = errors.New("ADD_OVERFLOW")
	ErrSubOverflow  = errors.New("SUB_OVERFLOW")
	ErrMulOverflow  = errors.New("MUL_OVERFLOW")
	ErrZeroDivision = errors.New("ZERO_DIVISION")

	OneE18              = uint256.NewInt(1e18) // 18 decimal places
	TwoE18              = uint256.NewInt(2e18)
	FourE18             = uint256.NewInt(4e18)
	MaxPowRelativeError = uint256.NewInt(10000) // 10^(-14)
)

var FixPoint *fixPoint

type fixPoint struct{}

func (f *fixPoint) MulDivUp(a, b, c *uint256.Int) (*uint256.Int, error) {
	return v3Utils.MulDivRoundingUp(a, b, c)
}

func (f *fixPoint) MulUp(a, b *uint256.Int) (*uint256.Int, error) {
	return v3Utils.MulDivRoundingUp(a, b, OneE18)
}

func (f *fixPoint) MulDown(a, b *uint256.Int) (*uint256.Int, error) {
	res, overflow := new(uint256.Int).MulDivOverflow(a, b, OneE18)
	if overflow {
		return nil, ErrMulOverflow
	}

	return res, nil
}

func (f *fixPoint) DivUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	return v3Utils.MulDivRoundingUp(a, OneE18, b)
}

func (f *fixPoint) DivDown(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	res, overflow := new(uint256.Int).MulDivOverflow(a, OneE18, b)
	if overflow {
		return nil, ErrMulOverflow
	}

	return res, nil
}

func (f *fixPoint) DivRawUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}
	var result uint256.Int
	v3Utils.DivRoundingUp(a, b, &result)
	return &result, nil
}

func (f *fixPoint) PowUp(x, y *uint256.Int) (*uint256.Int, error) {
	if y.Eq(OneE18) {
		return x, nil
	}

	if y.Eq(TwoE18) {
		return f.MulUp(x, x)
	}

	if y.Eq(FourE18) {
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
	maxError, err = f.MulUp(raw, MaxPowRelativeError)
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
	result := new(uint256.Int)
	if x.Lt(OneE18) {
		result.Sub(OneE18, x)
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
