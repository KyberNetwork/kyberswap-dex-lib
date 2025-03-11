package bignumber

import (
	"math"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

var (
	// TwoPow128 2^128
	TwoPow128 = new(uint256.Int).Lsh(uint256.NewInt(1), 128)
	Max       = new(uint256.Int).SetAllOne()

	ZeroBI = uint256.NewInt(0)
	One    = uint256.NewInt(1)
	Two    = uint256.NewInt(2)
	Three  = uint256.NewInt(3)
	Four   = uint256.NewInt(4)
	Five   = uint256.NewInt(5)
	U9     = uint256.NewInt(9)

	BasisPointUint256 = uint256.NewInt(10000)
)

var BONE = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18))
var BoneFloat, _ = new(big.Float).SetString("1000000000000000000")

var (
	preTenPowInt = lo.Map(lo.Range(18+1), func(n int, _ int) *uint256.Int {
		return uint256.NewInt(uint64(math.Pow10(n)))
	})
)

func TenPowInt(decimal uint8) *uint256.Int {
	if decimal <= 18 {
		return preTenPowInt[decimal]
	}
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
