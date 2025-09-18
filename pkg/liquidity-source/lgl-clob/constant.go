package lglclob

import (
	"errors"
	"math/big"
)

const (
	DexType = "lgl-clob"

	maxPriceLevels = 50
)

var (
	bMaxPriceLevels = big.NewInt(maxPriceLevels)

	ErrInvalidToken  = errors.New("invalid token")
	ErrInvalidAmount = errors.New("invalid amount")
	ErrEmptyOrders   = errors.New("empty orders")
)
