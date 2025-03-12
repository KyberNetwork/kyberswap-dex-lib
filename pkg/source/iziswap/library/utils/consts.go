package utils

import (
	"github.com/holiman/uint256"
)

var Pow96 = new(uint256.Int).Exp(uint256.NewInt(2), uint256.NewInt(96))
