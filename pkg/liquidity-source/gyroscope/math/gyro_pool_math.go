package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	ErrSqrtFailed = errors.New("_sqrt FAILED")
)

var GyroPoolMath *gyroPoolMath

var (
	number_1e2  = number.TenPow(2)
	number_1e3  = number.TenPow(3)
	number_1e4  = number.TenPow(4)
	number_1e5  = number.TenPow(5)
	number_1e6  = number.TenPow(6)
	number_1e7  = number.TenPow(7)
	number_1e8  = number.TenPow(8)
	number_1e9  = number.TenPow(9)
	number_1e10 = number.TenPow(10)
	number_1e11 = number.TenPow(11)
	number_1e12 = number.TenPow(12)
	number_1e13 = number.TenPow(13)
	number_1e14 = number.TenPow(14)
	number_1e15 = number.TenPow(15)
	number_1e16 = number.TenPow(16)
	number_1e17 = number.TenPow(17)
)

type gyroPoolMath struct {
	SQRT_1E_NEG_1  *uint256.Int
	SQRT_1E_NEG_3  *uint256.Int
	SQRT_1E_NEG_5  *uint256.Int
	SQRT_1E_NEG_7  *uint256.Int
	SQRT_1E_NEG_9  *uint256.Int
	SQRT_1E_NEG_11 *uint256.Int
	SQRT_1E_NEG_13 *uint256.Int
	SQRT_1E_NEG_15 *uint256.Int
	SQRT_1E_NEG_17 *uint256.Int
}

func init() {
	GyroPoolMath = &gyroPoolMath{
		SQRT_1E_NEG_1:  uint256.MustFromDecimal("316227766016837933"),
		SQRT_1E_NEG_3:  uint256.MustFromDecimal("31622776601683793"),
		SQRT_1E_NEG_5:  uint256.MustFromDecimal("3162277660168379"),
		SQRT_1E_NEG_7:  uint256.MustFromDecimal("316227766016837"),
		SQRT_1E_NEG_9:  uint256.MustFromDecimal("31622776601683"),
		SQRT_1E_NEG_11: uint256.MustFromDecimal("3162277660168"),
		SQRT_1E_NEG_13: uint256.MustFromDecimal("316227766016"),
		SQRT_1E_NEG_15: uint256.MustFromDecimal("31622776601"),
		SQRT_1E_NEG_17: uint256.MustFromDecimal("3162277660"),
	}
}

// Sqrt
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroPoolMath.sol#L121
func (l *gyroPoolMath) Sqrt(input, tolerance *uint256.Int) (*uint256.Int, error) {
	if input.IsZero() {
		return number.Zero, nil
	}

	guess := l._makeInitialGuess(input)

	// 7 iterations of newton's method
	guess = new(uint256.Int).Div(new(uint256.Int).Add(guess, new(uint256.Int).Div(new(uint256.Int).Mul(input, GyroFixedPoint.ONE), guess)), number.Number_2)
	guess = new(uint256.Int).Div(new(uint256.Int).Add(guess, new(uint256.Int).Div(new(uint256.Int).Mul(input, GyroFixedPoint.ONE), guess)), number.Number_2)
	guess = new(uint256.Int).Div(new(uint256.Int).Add(guess, new(uint256.Int).Div(new(uint256.Int).Mul(input, GyroFixedPoint.ONE), guess)), number.Number_2)
	guess = new(uint256.Int).Div(new(uint256.Int).Add(guess, new(uint256.Int).Div(new(uint256.Int).Mul(input, GyroFixedPoint.ONE), guess)), number.Number_2)
	guess = new(uint256.Int).Div(new(uint256.Int).Add(guess, new(uint256.Int).Div(new(uint256.Int).Mul(input, GyroFixedPoint.ONE), guess)), number.Number_2)
	guess = new(uint256.Int).Div(new(uint256.Int).Add(guess, new(uint256.Int).Div(new(uint256.Int).Mul(input, GyroFixedPoint.ONE), guess)), number.Number_2)
	guess = new(uint256.Int).Div(new(uint256.Int).Add(guess, new(uint256.Int).Div(new(uint256.Int).Mul(input, GyroFixedPoint.ONE), guess)), number.Number_2)

	guessSquared, err := GyroFixedPoint.MulDown(guess, guess)
	if err != nil {
		return nil, err
	}

	tmp, err := GyroFixedPoint.MulUp(guess, tolerance)
	if err != nil {
		return nil, err
	}

	upperBound, err := GyroFixedPoint.Add(input, tmp)
	if err != nil {
		return nil, err
	}

	lowerBound, err := GyroFixedPoint.Sub(input, tmp)
	if err != nil {
		return nil, err
	}

	if guessSquared.Gt(upperBound) || guessSquared.Lt(lowerBound) {
		return nil, ErrSqrtFailed
	}

	return guess, nil
}

// _makeInitialGuess
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroPoolMath.sol#L163
func (l *gyroPoolMath) _makeInitialGuess(input *uint256.Int) *uint256.Int {
	if input.Cmp(GyroFixedPoint.ONE) >= 0 {
		return new(uint256.Int).Mul(
			new(uint256.Int).Lsh(
				number.Number_1,
				l._intLog2Halved(new(uint256.Int).Div(input, GyroFixedPoint.ONE)),
			),
			GyroFixedPoint.ONE,
		)
	}

	if input.Cmp(number.Number_10) <= 0 {
		return l.SQRT_1E_NEG_17
	}

	if input.Cmp(number_1e2) <= 0 {
		return number_1e10
	}

	if input.Cmp(number_1e3) <= 0 {
		return l.SQRT_1E_NEG_15
	}

	if input.Cmp(number_1e4) <= 0 {
		return number_1e11
	}

	if input.Cmp(number_1e5) <= 0 {
		return l.SQRT_1E_NEG_13
	}

	if input.Cmp(number_1e6) <= 0 {
		return number_1e12
	}

	if input.Cmp(number_1e7) <= 0 {
		return l.SQRT_1E_NEG_11
	}

	if input.Cmp(number_1e8) <= 0 {
		return number_1e13
	}

	if input.Cmp(number_1e9) <= 0 {
		return l.SQRT_1E_NEG_9
	}

	if input.Cmp(number_1e10) <= 0 {
		return number_1e14
	}

	if input.Cmp(number_1e11) <= 0 {
		return l.SQRT_1E_NEG_7
	}

	if input.Cmp(number_1e12) <= 0 {
		return number_1e15
	}

	if input.Cmp(number_1e13) <= 0 {
		return l.SQRT_1E_NEG_5
	}

	if input.Cmp(number_1e14) <= 0 {
		return number_1e16
	}

	if input.Cmp(number_1e15) <= 0 {
		return l.SQRT_1E_NEG_3
	}

	if input.Cmp(number_1e16) <= 0 {
		return number_1e17
	}

	if input.Cmp(number_1e17) <= 0 {
		return l.SQRT_1E_NEG_1
	}

	return input
}

// _intLog2Halved
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroPoolMath.sol#L222C14-L222C28
func (l *gyroPoolMath) _intLog2Halved(x *uint256.Int) (n uint) {
	if !x.Lt(new(uint256.Int).Lsh(number.Number_1, 128)) {
		x = new(uint256.Int).Rsh(x, 128)
		n += 64
	}

	if !x.Lt(new(uint256.Int).Lsh(number.Number_1, 64)) {
		x = new(uint256.Int).Rsh(x, 64)
		n += 32
	}

	if !x.Lt(new(uint256.Int).Lsh(number.Number_1, 32)) {
		x = new(uint256.Int).Rsh(x, 32)
		n += 16
	}

	if !x.Lt(new(uint256.Int).Lsh(number.Number_1, 16)) {
		x = new(uint256.Int).Rsh(x, 16)
		n += 8
	}

	if !x.Lt(new(uint256.Int).Lsh(number.Number_1, 8)) {
		x = new(uint256.Int).Rsh(x, 8)
		n += 4
	}

	if !x.Lt(new(uint256.Int).Lsh(number.Number_1, 4)) {
		x = new(uint256.Int).Rsh(x, 4)
		n += 2
	}

	if !x.Lt(new(uint256.Int).Lsh(number.Number_1, 2)) {
		// x = new(uint256.Int).Rsh(x, 2)
		n += 1
	}

	return
}
