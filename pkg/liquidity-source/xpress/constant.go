package xpress

import (
	"errors"
	"math/big"
)

const (
	DexType = "xpress"

	DefaultGas = 200000

	maxPriceLevels = 50
)

var (
	bMaxPriceLevels = big.NewInt(maxPriceLevels)

	ErrInvalidToken  = errors.New("invalid token")
	ErrInvalidAmount = errors.New("invalid amount")
)
