package smardex

import "errors"

var (
	ErrZeroAmount            = errors.New("invalid zero amount")
	ErrSameAddress           = errors.New("invalid token in and token out are identical")
	ErrInvalidTimestamp      = errors.New("current timestamp is greater than priceAverageLastTimestamp")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
