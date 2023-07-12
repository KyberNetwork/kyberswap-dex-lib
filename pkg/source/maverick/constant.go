package maverick

import (
	"math/big"
	"time"
)

const (
	DexTypeMaverick       = "maverick"
	graphQLRequestTimeout = 20 * time.Second
)

var (
	zeroBI     = big.NewInt(0)
	zeroString = "0"
)
