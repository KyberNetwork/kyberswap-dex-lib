package wildcard

import (
	"errors"
)

const (
	DexType    = "wildcard"
	defaultGas = 120248
	sampleSize = 15
	bps        = 10000
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
