package two

import "errors"

var (
	ErrDenominatorZero = errors.New("denominator should not be 0")
)
