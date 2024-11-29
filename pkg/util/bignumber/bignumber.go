package bignumber

import (
	"math"
	"math/big"

	"github.com/samber/lo"
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

	MIN_SQRT_RATIO    = big.NewInt(4295128739)
	MAX_SQRT_RATIO, _ = new(big.Int).SetString("1461446703485210103287273052203988822378723970342", 10)

	BasisPoint = big.NewInt(10000)

	MAX_UINT_128 = new(big.Int).Sub(new(big.Int).Lsh(One, 128), One)
)

var BONE = new(big.Int).Exp(Ten, big.NewInt(18), nil)
var BoneFloat, _ = new(big.Float).SetString("1000000000000000000")

var (
	preTenPowDecimals = lo.Map(lo.Range(18+1), func(n int, _ int) *big.Float {
		return big.NewFloat(math.Pow10(n))
	})
	preTenPowInt = lo.Map(lo.Range(18+1), func(n int, _ int) *big.Int {
		return big.NewInt(int64(math.Pow10(n)))
	})
)

// TenPowDecimals calculates 10^decimal
func TenPowDecimals[T constraints.Integer](decimal T) *big.Float {
	if decimal <= 18 {
		return preTenPowDecimals[decimal]
	}
	return big.NewFloat(math.Pow10(int(decimal)))
}

func TenPowInt[T constraints.Integer](decimal T) *big.Int {
	if decimal <= 18 {
		return preTenPowInt[decimal]
	}
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

func Cap(n *big.Int, min *big.Int, max *big.Int) *big.Int {
	if n.Cmp(min) <= 0 {
		return new(big.Int).Add(min, One)
	}
	if n.Cmp(max) >= 0 {
		return new(big.Int).Sub(max, One)
	}
	return n
}

func CapPriceLimit(n *big.Int) *big.Int {
	return Cap(n, MIN_SQRT_RATIO, MAX_SQRT_RATIO)
}
