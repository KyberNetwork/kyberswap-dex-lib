package nomiswapstable

import (
	"github.com/holiman/uint256"
)

var (
	defaultGas  = Gas{Swap: 300000}
	A_PRECISION = uint256.NewInt(100)
	MaxFee      = uint256.NewInt(100000)
	Zero        = uint256.NewInt(0)
	One         = uint256.NewInt(1)
	Two         = uint256.NewInt(2)
	Three       = uint256.NewInt(3)
	Four        = uint256.NewInt(4)
)

const (
	MAX_LOOP_LIMIT = 256
)
