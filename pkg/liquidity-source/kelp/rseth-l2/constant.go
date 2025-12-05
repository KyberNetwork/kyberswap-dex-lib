package rsethl2

import (
	"errors"
	"math/big"
)

const (
	DexType        = "kelp-rseth-l2"
	defaultReserve = "100000000000000000000000000"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	BasisPoint      = big.NewInt(10000)
	ONE             = big.NewInt(1e18)
)
