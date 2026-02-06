package kipseliprop

import "errors"

const (
	DexType    = "kipseli-prop"
	defaultGas = 125_000
	sampleSize = 15
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
