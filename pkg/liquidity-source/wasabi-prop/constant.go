package wasabiprop

import "errors"

const (
	DexType    = "wasabi-prop"
	defaultGas = 200_000
	sampleSize = 15
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
