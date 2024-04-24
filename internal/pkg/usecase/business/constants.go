package business

import "errors"

var (
	ErrorNoPrice        = errors.New("no price for token")
	ErrorInvalidReserve = errors.New("invalid pool reserve")
	ErrNilLiquidity     = errors.New("liquidity is nil")
	ErrNilSqrtPrice     = errors.New("sqrtPrice is nil")
)
