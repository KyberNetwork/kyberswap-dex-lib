package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	ErrAddOverflow  = errors.New("ADD_OVERFLOW")
	ErrSubOverflow  = errors.New("SUB_OVERFLOW")
	ErrZeroDivision = errors.New("ZERO_DIVISION")
	ErrDivInternal  = errors.New("DIV_INTERNAL")
	ErrMulOverflow  = errors.New("MUL_OVERFLOW")
)

var GyroFixedPoint *gyroFixedPoint

type gyroFixedPoint struct {
	ONE        *uint256.Int
	MIDDECIMAL *uint256.Int
}

func init() {
	GyroFixedPoint = &gyroFixedPoint{
		ONE:        number.Number_1e18,
		MIDDECIMAL: uint256.NewInt(1e9),
	}
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L25
func (l *gyroFixedPoint) Add(a, b *uint256.Int) (*uint256.Int, error) {
	c := new(uint256.Int).Add(a, b)

	if c.Lt(a) {
		return nil, ErrAddOverflow
	}

	return c, nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L35
func (l *gyroFixedPoint) Sub(a, b *uint256.Int) (*uint256.Int, error) {
	if a.Lt(b) {
		return nil, ErrSubOverflow
	}

	return new(uint256.Int).Sub(a, b), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L59
func (l *gyroFixedPoint) MulUp(a, b *uint256.Int) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)

	if !(a.IsZero() || new(uint256.Int).Div(product, a).Eq(b)) {
		return nil, ErrMulOverflow
	}

	if product.IsZero() {
		return number.Zero, nil
	}

	return new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Sub(product, number.Number_1), l.ONE), number.Number_1), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L78
func (l *gyroFixedPoint) MulUpU(a, b *uint256.Int) *uint256.Int {
	product := new(uint256.Int).Mul(a, b)

	if product.IsZero() {
		return number.Zero
	}

	return new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Sub(product, number.Number_1), l.ONE), number.Number_1)
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L45
func (l *gyroFixedPoint) MulDown(a, b *uint256.Int) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)

	if !(a.IsZero() || new(uint256.Int).Div(product, a).Eq(b)) {
		return nil, ErrMulOverflow
	}

	return new(uint256.Int).Div(product, l.ONE), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L55
func (l *gyroFixedPoint) MulDownU(a, b *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(new(uint256.Int).Mul(a, b), l.ONE)
}

func (l *gyroFixedPoint) MulDownLargeSmallU(a, b *uint256.Int) *uint256.Int {
	return new(uint256.Int).Add(
		new(uint256.Int).Mul(new(uint256.Int).Div(a, l.ONE), b),
		l.MulDownU(new(uint256.Int).Mod(a, l.ONE), b),
	)
	// return (a / ONE) * b + mulDownU(a % ONE, b);
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L118
func (l *gyroFixedPoint) DivUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return number.Zero, nil
	}

	aInflated := new(uint256.Int).Mul(a, l.ONE)

	if !(new(uint256.Int).Div(aInflated, a).Eq(l.ONE)) {
		return nil, ErrDivInternal
	}

	return new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Sub(aInflated, number.Number_1), b), number.Number_1), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L141
func (l *gyroFixedPoint) DivUpU(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return number.Zero, nil
	}

	return new(uint256.Int).Add(
		new(uint256.Int).Div(
			new(uint256.Int).Sub(
				new(uint256.Int).Mul(a, l.ONE),
				number.Number_1,
			),
			b,
		),
		number.Number_1,
	), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L93
func (l *gyroFixedPoint) DivDown(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return number.Zero, nil
	}

	aInflated := new(uint256.Int).Mul(a, l.ONE)

	if !(new(uint256.Int).Div(aInflated, a).Eq(l.ONE)) {
		return nil, ErrDivInternal
	}

	return new(uint256.Int).Div(aInflated, b), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L110
func (l *gyroFixedPoint) DivDownU(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	return new(uint256.Int).Div(new(uint256.Int).Mul(a, l.ONE), b), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L180
func (l *gyroFixedPoint) DivDownLargeU(a, b *uint256.Int) (*uint256.Int, error) {
	return l.DivDownLargeU_2(a, b, l.MIDDECIMAL, l.MIDDECIMAL)
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L204
func (l *gyroFixedPoint) DivDownLargeU_2(a, b, d, e *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		// In this case only, the denominator of the outer division is zero, and we revert
		return nil, ErrZeroDivision
	}

	denom := new(uint256.Int).Add(number.Number_1, new(uint256.Int).Div(new(uint256.Int).Sub(b, number.Number_1), e))

	return new(uint256.Int).Div(
		new(uint256.Int).Mul(a, d),
		denom,
	), nil
}
