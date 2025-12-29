package tessera

import (
	"errors"
	"math/big"
)

const (
	DexType = "tessera"

	defaultGas = 400000
)

var (
	Thousand           = big.NewInt(1000)
	ErrInvalidToken    = errors.New("invalid token")
	ErrTradingDisabled = errors.New("trading disabled")
	ErrNotInitialised  = errors.New("pool not initialised")
	ErrInvalidRate     = errors.New("invalid rate")
	ErrInvalidSwapRate = errors.New("invalid swap rate")
	ErrSwapReverted    = errors.New("swap would revert")
	ErrInternal        = errors.New("internal error")
)
