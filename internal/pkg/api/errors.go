package api

import (
	"errors"
)

var (
	ErrBindQueryFailed       = errors.New("unable to bind query parameters")
	ErrBindRequestBodyFailed = errors.New("unable to bind request body")

	ErrInvalidRoute                  = errors.New("invalid route")
	ErrInvalidValue                  = errors.New("invalid value")
	ErrFeeAmountGreaterThanAmountIn  = errors.New("feeAmount is greater than amountIn")
	ErrFeeAmountGreaterThanAmountOut = errors.New("feeAmount is greater than amountOut")
)
