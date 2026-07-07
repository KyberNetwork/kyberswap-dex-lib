package sweth

import (
	"errors"
)

const (
	DexType = "swell-sweth"
)

const (
	// unlimited reserve
	reserves = "10000000000000000000"
)

var (
	ErrUnsupportedSwap = errors.New("unsupported swap")
	ErrPaused          = errors.New("paused")
)
