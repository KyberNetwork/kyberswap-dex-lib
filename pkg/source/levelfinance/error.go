package levelfinance

import "errors"

var (
	ErrSameTokenSwap = errors.New("same token swap")
	ErrZeroAmount    = errors.New("zero amount")
	ErrTokenNotFound = errors.New("token is not found")
)
