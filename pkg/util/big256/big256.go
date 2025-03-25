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

	MinSqrtRatio    = uint256.NewInt(4295128739)
	MaxSqrtRatio, _ = NewUint256("1461446703485210103287273052203988822378723970342")

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

func Cap(n *uint256.Int, min *uint256.Int, max *uint256.Int) *uint256.Int {
	if n.Cmp(min) <= 0 {
		return new(uint256.Int).Add(min, One)
	}
	if n.Cmp(max) >= 0 {
		return new(uint256.Int).Sub(max, One)
	}
	return n
}

func CapPriceLimit(priceLimit *uint256.Int) *uint256.Int {
	if priceLimit.Cmp(MinSqrtRatio) <= 0 {
		return priceLimit.AddUint64(MinSqrtRatio, 1)
	}
	if priceLimit.Cmp(MaxSqrtRatio) >= 0 {
		return priceLimit.SubUint64(MaxSqrtRatio, 1)
	}
	return priceLimit
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
