package baseline

import "errors"

const (
	DexType        = "baseline"
	defaultGas     = 300000
	bTokenDecimals = 18

	methodGetQuoteState     = "getQuoteState"
	methodQuoteBuyExactIn   = "quoteBuyExactIn"
	methodQuoteBuyExactOut  = "quoteBuyExactOut"
	methodQuoteSellExactIn  = "quoteSellExactIn"
	methodQuoteSellExactOut = "quoteSellExactOut"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidAmountIn  = errors.New("invalid amount in")
	ErrInvalidAmountOut = errors.New("invalid amount out")
	ErrPoolNotFound     = errors.New("pool not found")
	ErrNoRate           = errors.New("no cached rate")
)
