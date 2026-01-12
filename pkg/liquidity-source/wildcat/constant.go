package wildcat

import (
	"errors"
	"math/big"
)

const (
	DexType    = "wildcat"
	defaultGas = 10000
	sampleSize = 15
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")

	buffer = big.NewInt(9995)
)
