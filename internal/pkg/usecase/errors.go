package usecase

import (
	"errors"
)

var (
	ErrPublicKeyNotFound         = errors.New("public key is not found")
	ErrRouteCacheNotFound        = errors.New("route cache not found")
	ErrRouteCacheExpired         = errors.New("route cache expired")
	ErrRouteCacheUnmarshalFailed = errors.New("failed to unmarshal route cache")
)

var (
	ErrPriceImpactIsGreaterThanEpsilon = errors.New("priceImpact is greater than epsilon")
	ErrPoolNotFound                    = errors.New("pool not found")
	ErrInvalidSwap                     = errors.New("invalid swap")
	ErrFeeAmountIsGreaterThanAmountOut = errors.New("feeAmount is greater than amountOut")
	ErrAmountInIsGreaterThanMaxAllowed = errors.New("amountIn is greater than max allowed")
	ErrRouteNotFound                   = errors.New("route not found")
)
