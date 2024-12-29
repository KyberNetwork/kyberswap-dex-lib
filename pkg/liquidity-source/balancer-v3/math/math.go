package math

import (
	"github.com/holiman/uint256"
)

func MulDivUp(a, b, c *uint256.Int) (*uint256.Int, error) {
	if c.IsZero() {
		return nil, ErrZeroDivision
	}

	product, err := Mul(a, b)
	if err != nil {
		return nil, err
	}

	// Equivalent to:
	// result = a == 0 ? 0 : (a * b - 1) / c + 1
	if product.IsZero() {
		return ZERO, nil
	}

	product.Sub(product, ONE)
	product.Div(product, c)
	product.Add(product, ONE)

	return product, nil
}

func MulUp(a, b *uint256.Int) (*uint256.Int, error) {
	product, err := Mul(a, b)
	if err != nil {
		return nil, ErrMulOverflow
	}

	// Equivalent to:
	// result = product == 0 ? 0 : ((product - 1) / FixedPoint.ONE) + 1
	if product.IsZero() {
		return ZERO, nil
	}

	product.Sub(product, ONE)
	product.Div(product, ONE_E18)
	product.Add(product, ONE)

	return product, nil
}

func MulDown(a, b *uint256.Int) (*uint256.Int, error) {
	product, err := Mul(a, b)
	if err != nil {
		return nil, ErrMulOverflow
	}

	return product.Div(product, ONE_E18), nil
}

func DivUp(a, b *uint256.Int) (*uint256.Int, error) {
	return MulDivUp(a, ONE_E18, b)
}

func DivDown(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	aInflated, err := Mul(a, ONE_E18)
	if err != nil {
		return nil, err
	}

	return aInflated.Div(aInflated, b), nil
}

func Complement(x *uint256.Int) *uint256.Int {
	// Equivalent to:
	// result = (x < ONE) ? (ONE - x) : 0
	result := new(uint256.Int).Set(ZERO)
	if x.Lt(ONE_E18) {
		result.Sub(ONE_E18, x)
	}

	return result
}

func Add(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c, overflow := new(uint256.Int).AddOverflow(a, b)
	if overflow {
		return nil, ErrAddOverflow
	}
	return c, nil
}

func Sub(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c, overflow := new(uint256.Int).SubOverflow(a, b)
	if overflow {
		return nil, ErrSubOverflow
	}
	return c, nil
}

func Mul(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c, overflow := new(uint256.Int).MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	}
	return c, nil
}
