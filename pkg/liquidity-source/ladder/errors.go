package ladder

import "errors"

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrZeroAmountIn          = errors.New("zero amount in")
	ErrNoQuote               = errors.New("no quote available for direction")
	ErrAmountInTooLarge      = errors.New("amount in exceeds snapshot ladder")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
