package utils

import (
	"github.com/holiman/uint256"
)

type State struct {
	LiquidityX   *uint256.Int
	Liquidity    *uint256.Int
	CurrentPoint int
	SqrtPrice96  *uint256.Int
}
