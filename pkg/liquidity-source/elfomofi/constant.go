package elfomofi

import (
	"errors"
)

const (
	DexType    = "elfomofi"
	defaultGas = 308000
	sampleSize = 15
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
