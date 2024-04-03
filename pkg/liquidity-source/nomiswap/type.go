package nomiswap

import (
	"github.com/holiman/uint256"
)

type ExtraStablePool struct {
	SwapFee                   uint32       `json:"swapFee"`
	Token0PrecisionMultiplier *uint256.Int `json:"token0PrecisionMultiplier"`
	Token1PrecisionMultiplier *uint256.Int `json:"token1PrecisionMultiplier"`
	A                         *uint256.Int `json:"a"`
}

type Metadata struct {
	Offset int `json:"offset"`
}
