package bignumber

import (
	"math"
	"math/big"
)

// TwoPow128 2^128
var TwoPow128 = new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil)

// TenPowDecimals calculates 10^decimal
func TenPowDecimals(decimal uint8) *big.Float {
	return big.NewFloat(math.Pow10(int(decimal)))
}

func TenPowInt(decimal uint8) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimal)), nil)
}
