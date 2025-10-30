package bignumber

import (
	"math"
	"math/big"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
)

var (
	TwoPow64  = new(uint256.Int).Lsh(U1, 64)
	TwoPow128 = new(uint256.Int).Lsh(U1, 128)
	UMax      = new(uint256.Int).SetAllOne()

	U0   = uint256.NewInt(0)
	U1   = uint256.NewInt(1)
	U2   = uint256.NewInt(2)
	U3   = uint256.NewInt(3)
	U4   = uint256.NewInt(4)
	U5   = uint256.NewInt(5)
	U6   = uint256.NewInt(6)
	U8   = uint256.NewInt(8)
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

func TenPow[T constraints.Integer](decimal T) *uint256.Int {
	if int(decimal) < len(preTenPow) {
		return preTenPow[decimal]
	}
	tmp := uint256.NewInt(uint64(decimal))
	return tmp.Exp(U10, tmp)
}

func New(s string) *uint256.Int {
	res, _ := uint256.FromDecimal(s)
	return res
}

func NewUint256(s string) (res *uint256.Int, err error) {
	res = new(uint256.Int)
	err = res.SetFromDecimal(s)
	return
}

func FromBig(big *big.Int) *uint256.Int {
	u, _ := uint256.FromBig(big)
	return u
}

func MustFromBigs[S ~[]*big.Int](bigs S) []*uint256.Int {
	return lo.Map(bigs, func(b *big.Int, _ int) *uint256.Int {
		return uint256.MustFromBig(b)
	})
}

func MustFromInt64(x int64) *uint256.Int {
	return uint256.MustFromBig(new(big.Int).SetInt64(x))
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

// MulDivUp multiplies x and y, then divides by denominator, rounding up, and stores the result in res.
func MulDivUp(res, x, y, denominator *uint256.Int) *uint256.Int {
	_ = v3Utils.MulDivRoundingUpV2(x, y, denominator, res)
	return res
}

// MulDivDown multiplies x and y, then divides by denominator, rounding down, and stores the result in res.
func MulDivDown(res, x, y, denominator *uint256.Int) *uint256.Int {
	res.MulDivOverflow(x, y, denominator)
	return res
}

// MulWadUp multiplies x and y, then divides by BONE, rounding up, and stores the result in res.
func MulWadUp(res, x, y *uint256.Int) *uint256.Int {
	return MulDivUp(res, x, y, BONE)
}

// MulWadDown multiplies x and y, then divides by BONE, rounding down, and stores the result in res.
func MulWadDown(res, x, y *uint256.Int) *uint256.Int {
	return MulDivDown(res, x, y, BONE)
}
