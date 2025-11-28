package orderbook

import (
	"math"
)

const (
	DexType = "orderbook"

	MaxAge = math.MaxInt64 // TODO: parametize this and gas
)

var (
	defaultGas = Gas{Base: 95643}
)
