package two

import "errors"

var (
	ErrDenominatorZero = errors.New("denominator should not be 0")
	ErrOverflow        = errors.New("uint256 overflow")
	ErrUnderflow       = errors.New("uint256 underflow")

	errUnsafeA            = errors.New("unsafe values A")
	errUnsafeGamma        = errors.New("unsafe values gamma")
	errUnsafeD            = errors.New("unsafe values D")
	errUnsafeX0           = errors.New("unsafe values x[0]")
	errUnsafeX            = errors.New("unsafe values x[i]")
	errUnsafeY            = errors.New("unsafe value for y")
	errDidNotConverge     = errors.New("did not converge")
	errSqrtDidNotConverge = errors.New("sqrt_int did not converge")
	errLoss               = errors.New("loss")

	errSameToken            = errors.New("tokenIn and tokenOut must not be the same")
	errTokenIndexOutOfRange = errors.New("token index is out of range")
	errSameCoin             = errors.New("i = j")
	errCoinIndexOutOfRange  = errors.New("coin index out of range")
	errZeroAmount           = errors.New("do not exchange 0 coins")
	errNegativeAmount       = errors.New("negative amount")
	errNotTwoCoins          = errors.New("curve-two pool must have 2 coins")
	errMissingPoolData      = errors.New("curve-two pool is missing reserves or precisions")
)
