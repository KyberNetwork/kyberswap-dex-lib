package math

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var FixPoint *fixPoint

type fixPoint struct{}

func (f *fixPoint) MulDivUp(a, b, c *uint256.Int) (*uint256.Int, error) {
	return v3Utils.MulDivRoundingUp(a, b, c)
}

func (f *fixPoint) MulUp(a, b *uint256.Int) (*uint256.Int, error) {
	return v3Utils.MulDivRoundingUp(a, b, U1e18)
}

func (f *fixPoint) MulDown(a, b *uint256.Int) (*uint256.Int, error) {
	res, overflow := new(uint256.Int).MulDivOverflow(a, b, U1e18)
	if overflow {
		return nil, ErrMulOverflow
	}

	return res, nil
}

func (f *fixPoint) DivUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	return v3Utils.MulDivRoundingUp(a, U1e18, b)
}

func (f *fixPoint) DivDown(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	res, overflow := new(uint256.Int).MulDivOverflow(a, U1e18, b)
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
	if y.Eq(U1e18) {
		return x, nil
	}

	if y.Eq(U2e18) {
		return f.MulUp(x, x)
	}

	if y.Eq(U4e18) {
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
	maxError, err = f.MulUp(raw, UMaxPowRelativeError)
	if err != nil {
		return nil, err
	}

	maxError, err = f.Add(maxError, U1)
	if err != nil {
		return nil, err
	}

	return f.Add(raw, maxError)
}

func (f *fixPoint) Complement(x *uint256.Int) *uint256.Int {
	// result = (x < ONE) ? (ONE - x) : 0
	result := new(uint256.Int)
	if x.Lt(U1e18) {
		result.Sub(U1e18, x)
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
