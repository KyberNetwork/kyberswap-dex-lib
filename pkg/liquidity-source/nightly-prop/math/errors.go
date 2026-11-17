package math

import "errors"

var (
	// ErrOverflow is Solidity's panic 0x11 and SafeCast's overflow revert: a checked operation, a
	// cast, or a rounded-up result that does not fit its type.
	ErrOverflow = errors.New("nightly-prop math: arithmetic overflow")
	// ErrDivisionByZero is Solidity's panic 0x12.
	ErrDivisionByZero = errors.New("nightly-prop math: division by zero")
	// ErrInexactAdjustment is PriceMathInexactAdjustment: an offset or spread adjustment would
	// round.
	ErrInexactAdjustment = errors.New("nightly-prop math: inexact price adjustment")
	// ErrDenominatorMismatch is PriceMathDenominatorMismatch: two prices with different
	// denominators were combined.
	ErrDenominatorMismatch = errors.New("nightly-prop math: price denominator mismatch")
)
