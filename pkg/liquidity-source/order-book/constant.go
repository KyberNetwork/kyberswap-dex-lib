package orderbook

import (
	"time"
)

const (
	DexType = "orderbook"

	MaxAge = time.Minute
)

var (
	defaultGas = Gas{Base: 68331}
)
