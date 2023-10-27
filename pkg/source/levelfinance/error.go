package levelfinance

import "errors"

var (
	ErrSameTokenSwap       = errors.New("same token swap")
	ErrZeroAmount          = errors.New("zero amount")
	ErrTokenInfoIsNotFound = errors.New("token info is not found")
)
