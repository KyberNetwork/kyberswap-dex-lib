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

type gyroPoolMath struct {
}

func init() {
	GyroPoolMath = &gyroPoolMath{}
}

// Sqrt
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroPoolMath.sol#L121
func (l *gyroPoolMath) Sqrt(input, tolerance *uint256.Int) (*uint256.Int, error) {
	if input.Eq(number.Zero) {
		return number.Zero, nil
	}

	guess := l._makeInitialGuess(input)

	//guess = (guess + ((input * GyroFixedPoint.ONE) / guess)) / 2
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

	return guessSquared, nil
}

// _makeInitialGuess
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroPoolMath.sol#L163
func (l *gyroPoolMath) _makeInitialGuess(input *uint256.Int) *uint256.Int {
	// TODO: implement
	return nil
}

// _intLog2Halved
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroPoolMath.sol#L222C14-L222C28
func (l *gyroPoolMath) _intLog2Halved(x *uint256.Int) *uint256.Int {
	// TODO: implement
	return nil
}
