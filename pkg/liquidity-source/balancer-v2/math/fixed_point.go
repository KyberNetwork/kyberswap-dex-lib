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

var FixedPoint *fixedPoint

type fixedPoint struct {
	ZERO                   *uint256.Int
	ONE                    *uint256.Int
	TWO                    *uint256.Int
	FOUR                   *uint256.Int
	MAX_POW_RELATIVE_ERROR *uint256.Int
}

func init() {
	zero := uint256.NewInt(0)
	one := number.Number_1e18
	two := new(uint256.Int).Mul(number.Number_2, one)
	four := new(uint256.Int).Mul(number.Number_4, one)

	FixedPoint = &fixedPoint{
		ZERO:                   zero,
		ONE:                    one,
		TWO:                    two,
		FOUR:                   four,
		MAX_POW_RELATIVE_ERROR: uint256.NewInt(10000),
	}
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/FixedPoint.sol#L34
func (l *fixedPoint) Add(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c := new(uint256.Int).Add(a, b)

	if c.Lt(a) {
		return nil, ErrAddOverflow
	}

	return c, nil
}

func (l *fixedPoint) Sub(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	if a.Lt(b) {
		return nil, ErrSubOverflow
	}

	return new(uint256.Int).Sub(a, b), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/FixedPoint.sol#L83
func (l *fixedPoint) DivUp(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	aInflated := new(uint256.Int).Mul(a, l.ONE)

	if !(a.IsZero() || new(uint256.Int).Div(aInflated, a).Eq(l.ONE)) {
		return nil, ErrDivInternal
	}

	if aInflated.IsZero() {
		return number.Zero, nil
	}

	return new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Sub(aInflated, number.Number_1), b), number.Number_1), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/FixedPoint.sol#L74
func (l *fixedPoint) DivDown(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	aInflated := new(uint256.Int).Mul(a, l.ONE)

	if !(a.IsZero() || new(uint256.Int).Div(aInflated, a).Eq(l.ONE)) {
		return nil, ErrDivInternal
	}

	return new(uint256.Int).Div(aInflated, b), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/FixedPoint.sol#L132
func (l *fixedPoint) PowUp(x *uint256.Int, y *uint256.Int) (*uint256.Int, error) {
	if y.Eq(l.ONE) {
		return x, nil
	}
	if y.Eq(l.TWO) {
		return l.MulUp(x, x)
	}
	if y.Eq(l.FOUR) {
		square, err := l.MulUp(x, x)
		if err != nil {
			return nil, err
		}

		return l.MulUp(square, square)
	}

	raw, err := LogExpMath.Pow(x, y)
	if err != nil {
		return nil, err
	}

	mulUpRawAndMaxPow, err := l.MulUp(raw, l.MAX_POW_RELATIVE_ERROR)
	if err != nil {
		return nil, err
	}

	maxError, err := l.Add(mulUpRawAndMaxPow, number.Number_1)
	if err != nil {
		return nil, err
	}

	return l.Add(raw, maxError)
}

func (l *fixedPoint) PowUpV1(x *uint256.Int, y *uint256.Int) (*uint256.Int, error) {
	raw, err := LogExpMath.Pow(x, y)
	if err != nil {
		return nil, err
	}

	mulUpRawAndMaxPow, err := l.MulUp(raw, l.MAX_POW_RELATIVE_ERROR)
	if err != nil {
		return nil, err
	}

	maxError, err := l.Add(mulUpRawAndMaxPow, number.Number_1)
	if err != nil {
		return nil, err
	}

	return l.Add(raw, maxError)
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/FixedPoint.sol#L50
func (l *fixedPoint) MulDown(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)

	if !(a.IsZero() || new(uint256.Int).Div(product, a).Eq(b)) {
		return nil, ErrMulOverflow
	}

	return new(uint256.Int).Div(product, l.ONE), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/FixedPoint.sol#L57
func (l *fixedPoint) MulUp(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)

	if !(a.IsZero() || new(uint256.Int).Div(product, a).Eq(b)) {
		return nil, ErrMulOverflow
	}

	if product.IsZero() {
		return number.Zero, nil
	}

	return new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Sub(product, number.Number_1), l.ONE), number.Number_1), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/FixedPoint.sol#L156
func (l *fixedPoint) Complement(x *uint256.Int) *uint256.Int {
	if x.Lt(l.ONE) {
		return new(uint256.Int).Sub(l.ONE, x)
	}

	return number.Zero
}
