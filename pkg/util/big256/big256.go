package bignumber

import (
	"math"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

var (
	// TwoPow128 2^128
	TwoPow128 = new(uint256.Int).Lsh(U1, 128)
	UMax      = new(uint256.Int).SetAllOne()

	U0   = uint256.NewInt(0)
	U1   = uint256.NewInt(1)
	U2   = uint256.NewInt(2)
	U3   = uint256.NewInt(3)
	U4   = uint256.NewInt(4)
	U5   = uint256.NewInt(5)
	U6   = uint256.NewInt(6)
	U9   = uint256.NewInt(9)
	U10  = uint256.NewInt(10)
	U100 = uint256.NewInt(100)

	MinSqrtRatio    = uint256.NewInt(4295128739)
	MaxSqrtRatio, _ = NewUint256("1461446703485210103287273052203988822378723970342")

	UBasisPoint = uint256.NewInt(10000)
)

var BONE = TenPow(18)
var BoneFloat, _ = new(big.Float).SetString("1000000000000000000")

var (
	preTenPow = lo.Map(lo.Range(36+1), func(n int, _ int) *uint256.Int {
		if n < 20 {
			return uint256.NewInt(uint64(math.Pow10(n)))
		}
		tmp := uint256.NewInt(uint64(n))
		return tmp.Exp(U10, tmp)
	})
)

func TenPow(decimal uint8) *uint256.Int {
	if int(decimal) < len(preTenPow) {
		return preTenPow[decimal]
	}
	tmp := uint256.NewInt(uint64(decimal))
	return tmp.Exp(U10, tmp)
}

func NewUint256(s string) (res *uint256.Int, err error) {
	res = new(uint256.Int)
	err = res.SetFromDecimal(s)
	return
}

func Cap(n *uint256.Int, min *uint256.Int, max *uint256.Int) *uint256.Int {
	if n.Cmp(min) <= 0 {
		return new(uint256.Int).Add(min, U1)
	}
	if n.Cmp(max) >= 0 {
		return new(uint256.Int).Sub(max, U1)
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
	} else if a.Cmp(b) < 0 {
		return a
	}
	return b
}

// Max returns the larger of a or b.
func Max(a, b *uint256.Int) *uint256.Int {
	if a == nil || b == nil {
		return nil
	} else if a.Cmp(b) > 0 {
		return a
	}
	return b
}
