package kyberpmm

import "errors"

var (
	ErrTokenNotFound          = errors.New("token not found")
	ErrNoPriceLevelsForPool   = errors.New("no price levels for pool")
	ErrEmptyPriceLevels       = errors.New("empty price levels")
	ErrInsufficientLiquidity  = errors.New("insufficient liquidity")
	ErrInvalidFirmQuoteParams = errors.New("invalid firm quote params")
)
