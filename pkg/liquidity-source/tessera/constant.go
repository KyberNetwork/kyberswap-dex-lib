package tessera

import (
	"errors"
)

const (
	DexType = "tessera"

	defaultGas = 400000
)

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrTradingDisabled        = errors.New("trading disabled")
	ErrNotInitialised         = errors.New("pool not initialised")
	ErrInvalidRate            = errors.New("invalid rate")
	ErrSwapReverted           = errors.New("swap would revert")
)
