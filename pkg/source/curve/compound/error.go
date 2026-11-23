package compound

import "errors"

var (
	ErrZero                         = errors.New("zero")
	ErrBalancesMustMatchMultipliers = errors.New("balances must match multipliers")
	ErrDDoesNotConverge             = errors.New("d does not converge")
	ErrTokenFromEqualsTokenTo       = errors.New("can't compare token to itself")
	ErrTokenIndexesOutOfRange       = errors.New("token index out of range")
	ErrAmountOutNotConverge         = errors.New("approximation did not converge")
	ErrDenominatorZero              = errors.New("denominator should not be 0")
	// ErrNegativeResult: legacy returned a negative value (or drove a reserve negative); uint256 cannot hold it.
	ErrNegativeResult = errors.New("negative result")
)
