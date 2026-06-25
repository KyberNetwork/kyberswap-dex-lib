package ambient

import (
	"math/bits"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const q48u64 = uint64(1) << 48

func ApproxSqrtCompound(x64 uint64) uint64 {
	if x64 >= q48u64 {
		x64 = q48u64 - 1
	}
	hi, lo := bits.Mul64(x64, x64)
	xSq := (hi << 16) | (lo >> 48)
	linear := x64 >> 1
	quad := xSq >> 3
	return linear - quad
}

func CompoundStack(x, y uint64) uint64 {
	a := q48u64 + x
	b := q48u64 + y
	hi, lo := bits.Mul64(a, b)
	if hi>>48 != 0 {
		return ^uint64(0)
	}
	term := (hi << 16) | (lo >> 48)
	return term - q48u64
}

func CompoundShrink(val, deflator uint64) uint64 {
	q, _ := bits.Div64(val>>16, val<<48, q48u64+deflator)
	return q
}

// CompoundDivide computes floor(inflated * 2^48 / seed) - 2^48, capped at 2^48.
func CompoundDivide(inflated, seed *uint256.Int) uint64 {
	var num uint256.Int
	num.Lsh(inflated, 48)
	num.Div(&num, seed)
	num.Sub(&num, uQ48)
	if num.Gt(uQ48) {
		return uQ48.Uint64()
	}
	return num.Uint64()
}

// CompoundPrice applies compound growth to a Q64 sqrt price.
// shiftUp=true:  dst = price * (Q48+growth) >> 48 + 1
// shiftUp=false: dst = price << 48 / (Q48+growth)
func CompoundPrice(dst, price *uint256.Int, growth uint64, shiftUp bool) *uint256.Int {
	var multFactor uint256.Int
	multFactor.SetUint64(growth)
	multFactor.Add(&multFactor, uQ48)
	if shiftUp {
		dst.Mul(price, &multFactor)
		dst.Rsh(dst, 48)
		return dst.Add(dst, u256.U1)
	}
	dst.Lsh(price, 48)
	return dst.Div(dst, &multFactor)
}

// InflateLiqSeed sets dst = floor(seed * (Q48+growth) / Q48), capped at 2^128-1.
func InflateLiqSeed(dst, seed *uint256.Int, growth uint64) *uint256.Int {
	var multFactor uint256.Int
	multFactor.SetUint64(growth)
	multFactor.Add(&multFactor, uQ48)
	dst.Mul(seed, &multFactor)
	dst.Rsh(dst, 48)
	if dst.Gt(u256.UMaxU128) {
		dst.Set(u256.UMaxU128)
	}
	return dst
}
