package kokonutcrypto

import "errors"

var (
	ErrIndexOutOfRange   = errors.New("coin index out of range")
	ErrDenominatorZero   = errors.New("denominator should not be 0")
	ErrDySmallerThanZero = errors.New("dy is smaller than zero")
	ErrUnsafeValueY      = errors.New("unsafe values Y")
	ErrUnsafeValueD      = errors.New("unsafe values D")
	ErrUnsafeValuesGamma = errors.New("unsafe values gamma")
	ErrUnsafeValuesA     = errors.New("unsafe values A")
	ErrUnsafeValuesXi    = errors.New("unsafe values x[i]")
	ErrDidNotCoverage    = errors.New("did not coverage")
	ErrK0                = errors.New("k0")
	ErrD                 = errors.New("D")
	ErrLoss              = errors.New("loss")
)
