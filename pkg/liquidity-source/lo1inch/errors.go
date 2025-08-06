package lo1inch

import "errors"

var (
	ErrTokenInNotSupported       = errors.New("tokenIn is not supported")
	ErrNoOrderAvailable          = errors.New("no order available")
	ErrCannotFulfillAmountIn     = errors.New("cannot fulfill amountIn")
	ErrOnlyAllowFullFill         = errors.New("order only allow full fill")
	ErrFeeTakerExtensionNotFound = errors.New("fee taker extension not found")
)
