package bignumber

import (
	"fmt"
	"math"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
)

var (
	ZeroBI     = big.NewInt(0)
	One        = big.NewInt(1)
	Two        = big.NewInt(2)
	Three      = big.NewInt(3)
	Four       = big.NewInt(4)
	Five       = big.NewInt(5)
	Six        = big.NewInt(6)
	Seven      = big.NewInt(7)
	Eight      = big.NewInt(8)
	Nine       = big.NewInt(9)
	Ten        = big.NewInt(10)
	Eleven     = big.NewInt(11)
	B100       = big.NewInt(100)
	B2Pow24    = big.NewInt(1 << 24)
	B2Pow31    = big.NewInt(1 << 31)
	B2Pow128   = new(big.Int).Lsh(One, 128)
	B2Pow256   = new(big.Int).Lsh(One, 256)
	MaxUint128 = new(big.Int).Sub(B2Pow128, One)
	MaxUint256 = new(big.Int).Sub(B2Pow256, One)
	MinInt128  = new(big.Int).Neg(new(big.Int).Lsh(One, 127))
	MaxInt128  = new(big.Int).Sub(new(big.Int).Lsh(One, 127), One)

	MinSqrtRatio = big.NewInt(4295128739)
	MaxSqrtRatio = NewBig10("1461446703485210103287273052203988822378723970342")

	BasisPoint   = big.NewInt(10000)
	BasisPointM1 = big.NewInt(10000 - 1)
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
func TenPowDecimals[T constraints.Integer](decimals T) *big.Float {
	if 0 <= decimals && decimals <= 18 {
		return preTenPowDecimals[decimals]
	}
	return big.NewFloat(math.Pow10(int(decimals)))
}

func TenPowInt[T constraints.Integer](decimals T) *big.Int {
	if decimals < 0 {
		panic(fmt.Sprintf("decimals cannot be negative: %d", decimals))
	} else if decimals <= 18 {
		return preTenPowInt[decimals]
	}
	y := big.NewInt(int64(decimals))
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

func NewUint256(s string) (res *uint256.Int) {
	res = new(uint256.Int)
	_ = res.SetFromDecimal(s)
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
	return Cap(n, MinSqrtRatio, MaxSqrtRatio)
}

// Min returns the smaller of a or b.
func Min(a, b *big.Int) *big.Int {
	if a == nil || b == nil {
		return nil
	}

	if a.Cmp(b) < 0 {
		return a
	}

	return b
}

func ToStrings(vals []*big.Int) []string {
	return lo.Map(vals, func(v *big.Int, _ int) string {
		if v != nil {
			return v.String()
		}
		return "0"
	})
}

func Sum(vals []*big.Int) *big.Int {
	return lo.Reduce(vals, func(sum *big.Int, v *big.Int, _ int) *big.Int {
		if v != nil {
			return new(big.Int).Add(sum, v)
		}
		return sum
	}, new(big.Int))
}

// MulDivDown multiplies x and y, then divides by denominator, rounding down, and stores the result in res.
func MulDivDown(res, x, y, denominator *big.Int) *big.Int {
	return res.Mul(x, y).Quo(res, denominator)
}

// MulDivUp multiplies x and y, then divides by denominator, rounding up, and stores the result in res.
func MulDivUp(res, x, y, denominator *big.Int) *big.Int {
	var rem big.Int
	if res.Mul(x, y).QuoRem(res, denominator, &rem); rem.Sign() > 0 {
		res.Add(res, One)
	}
	return res
}
