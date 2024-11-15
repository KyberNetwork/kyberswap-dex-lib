package bignumber

import (
	"math"
	"math/big"

	"golang.org/x/exp/constraints"
)

var (
	// TwoPow128 2^128
	TwoPow128 = new(big.Int).Exp(Two, big.NewInt(128), nil)

	ZeroBI = big.NewInt(0)
	One    = big.NewInt(1)
	Two    = big.NewInt(2)
	Three  = big.NewInt(3)
	Four   = big.NewInt(4)
	Five   = big.NewInt(5)
	Six    = big.NewInt(6)
	Ten    = big.NewInt(10)
)

var BONE = new(big.Int).Exp(Ten, big.NewInt(18), nil)
var BoneFloat, _ = new(big.Float).SetString("1000000000000000000")

// TenPowDecimals calculates 10^decimal
func TenPowDecimals[T constraints.Integer](decimal T) *big.Float {
	return big.NewFloat(math.Pow10(int(decimal)))
}

func TenPowInt[T constraints.Integer](decimal T) *big.Int {
	y := big.NewInt(int64(decimal))
	return y.Exp(Ten, y, nil)
}

func NewBig10(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 10)
	return res
}

func NewBig(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 0)
	return res
}
