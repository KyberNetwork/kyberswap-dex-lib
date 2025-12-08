package rsethl2

import (
	"errors"
)

const (
	DexType        = "kelp-rseth-l2"
	defaultReserve = "100000000000000000000000000"
	defaultGas     = 76930
)

var (
	ErrInvalidToken = errors.New("invalid token")
)
