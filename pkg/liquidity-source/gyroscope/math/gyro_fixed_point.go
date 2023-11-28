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
	ONE *uint256.Int
}

func init() {
	one := number.Number_1e18

	GyroFixedPoint = &gyroFixedPoint{
		ONE: one,
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

	if !(a.Eq(number.Zero) || new(uint256.Int).Div(product, a).Eq(b)) {
		return nil, ErrMulOverflow
	}

	if product.Eq(number.Zero) {
		return number.Zero, nil
	}

	return new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Sub(product, number.Number_1), l.ONE), number.Number_1), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L93
func (l *gyroFixedPoint) DivDown(a, b *uint256.Int) (*uint256.Int, error) {
	if b.Eq(number.Zero) {
		return nil, ErrZeroDivision
	}

	if a.Eq(number.Zero) {
		return number.Zero, nil
	}

	aInflated := new(uint256.Int).Mul(a, l.ONE)

	if !(new(uint256.Int).Div(aInflated, a).Eq(l.ONE)) {
		return nil, ErrDivInternal
	}

	return new(uint256.Int).Div(aInflated, b), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L118
func (l *gyroFixedPoint) DivUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.Eq(number.Zero) {
		return nil, ErrZeroDivision
	}

	if a.Eq(number.Zero) {
		return number.Zero, nil
	}

	aInflated := new(uint256.Int).Mul(a, l.ONE)

	if !(new(uint256.Int).Div(aInflated, a).Eq(l.ONE)) {
		return nil, ErrDivInternal
	}

	return new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Sub(aInflated, number.Number_1), b), number.Number_1), nil
}

// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroFixedPoint.sol#L45
func (l *gyroFixedPoint) MulDown(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)

	if !(a.Eq(number.Zero) || new(uint256.Int).Div(product, a).Eq(b)) {
		return nil, ErrMulOverflow
	}

	return new(uint256.Int).Div(product, l.ONE), nil
}
