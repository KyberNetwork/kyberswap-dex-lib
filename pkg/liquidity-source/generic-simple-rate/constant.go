package generic_simple_rate

import (
	"errors"
)

const (
	DexType = "generic-simple-rate"

	defaultTokenWeight       = 1
	defaultReserves          = "100000000000000000000000000"
	DefaultGas         int64 = 60000
)

var (
	ErrPoolPaused = errors.New("pool is paused")
	ErrOverflow   = errors.New("overflow")
)
