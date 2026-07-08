package shared

import "errors"

var (
	ErrStaticExtraEmpty        = errors.New("staticExtra is empty")
	ErrExtraEmpty              = errors.New("extra is empty")
	ErrInvalidToken            = errors.New("invalid token")
	ErrBaseSwapAmountTooSmall  = errors.New("BASE_SWAP_AMOUNT_TOO_SMALL")
	ErrQuoteSwapAmountTooSmall = errors.New("QUOTE_SWAP_AMOUNT_TOO_SMALL")
)
