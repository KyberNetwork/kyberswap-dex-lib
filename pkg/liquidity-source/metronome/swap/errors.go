package metronomeswap

import "errors"

var (
	ErrSameToken             = errors.New("tokenIn and tokenOut are the same")
	ErrInvalidToken          = errors.New("token not found in pool")
	ErrSwapInactive          = errors.New("swap is inactive on this pool")
	ErrTokenInactive         = errors.New("synthetic token is inactive")
	ErrInvalidAmountIn       = errors.New("invalid amountIn")
	ErrZeroAmountOut         = errors.New("amountOut is zero after fee")
	ErrExceedsMaxTotalSupply = errors.New("amountOut would exceed syntheticTokenOut's maxTotalSupply")
)
