package business

import "errors"

var (
	ErrorInvalidReserve = errors.New("invalid pool reserve")
	ErrNilLiquidity     = errors.New("liquidity is nil")
	ErrNilSqrtPrice     = errors.New("sqrtPrice is nil")
)
