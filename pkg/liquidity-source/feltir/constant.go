package feltir

import "errors"

const (
	DexType    = "feltir"
	defaultGas = 200_000
	sampleSize = 15
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
