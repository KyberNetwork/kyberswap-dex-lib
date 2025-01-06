package bignumber

import (
	"math"
	"math/big"

	"github.com/holiman/uint256"
)

var (
	// TwoPow128 2^128
	TwoPow128 = new(uint256.Int).Exp(uint256.NewInt(2), uint256.NewInt(128))

	ZeroBI = uint256.NewInt(0)
	One    = uint256.NewInt(1)
	Two    = uint256.NewInt(2)
	Three  = uint256.NewInt(3)
	Four   = uint256.NewInt(4)
	Five   = uint256.NewInt(5)
	Six    = uint256.NewInt(6)
)

var BONE = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18))
var BoneFloat, _ = new(big.Float).SetString("1000000000000000000")

// TenPowDecimals calculates 10^decimal
func TenPowDecimals(decimal uint8) *big.Float {
	return big.NewFloat(math.Pow10(int(decimal)))
}

func TenPowInt(decimal uint8) *uint256.Int {
	return new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(uint64(decimal)))
}

func NewUint256(s string) (res *uint256.Int, err error) {
	res = new(uint256.Int)
	err = res.SetFromDecimal(s)
	return
}

// Min returns the smaller of a or b.
func Min(a, b *uint256.Int) *uint256.Int {
	if a == nil || b == nil {
		return nil
	}

	if a.Cmp(b) < 0 {
		return a
	}

	return b
}
