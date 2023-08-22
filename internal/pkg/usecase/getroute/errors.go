package getroute

import (
	"errors"
)

var (
	ErrInvalidSwap = errors.New("invalid swap")

	ErrPoolSetEmpty = errors.New("pool set is empty")

	ErrRouteNotFound = errors.New("route not found")

	ErrPriceImpactIsGreaterThanThreshold = errors.New("price impact is greater than threshold")

	ErrTokenNotFound = errors.New("token not found")

	ErrFeeAmountIsGreaterThanAmountOut = errors.New("feeAmount is greater than amountOut")

	ErrAmountInIsGreaterThanMaxAllowed = errors.New("amountIn is greater than max allowed")

	ErrNoTokenInPrice = errors.New("no token in price")
)
