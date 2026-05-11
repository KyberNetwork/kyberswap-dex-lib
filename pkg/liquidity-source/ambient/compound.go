package ambient

import (
	"math/big"
	"math/bits"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const q48u64 = uint64(1) << 48

func ApproxSqrtCompound(x64 uint64) uint64 {
	if x64 >= q48u64 {
		x64 = q48u64 - 1
	}
	hi, lo := bits.Mul64(x64, x64)
	xSq := (hi << 16) | (lo >> 48) // (x*x) >> 48, x < 2^48 so product < 2^96, fits 64b.
	linear := x64 >> 1
	quad := xSq >> 3
	return linear - quad
}

func CompoundStack(x, y uint64) uint64 {
	a := q48u64 + x
	b := q48u64 + y
	hi, lo := bits.Mul64(a, b)
	// term = (a*b) >> 48; if hi occupies more than 48 bits the result would
	// overflow uint64 after the minus-Q48 step, saturate at max.
	if hi>>48 != 0 {
		return ^uint64(0)
	}
	term := (hi << 16) | (lo >> 48)
	return term - q48u64
}

func CompoundShrink(val, deflator uint64) uint64 {
	// (val << 48) / (Q48 + deflator). val * 2^48 fits in 128 bits.
	q, _ := bits.Div64(val>>16, val<<48, q48u64+deflator)
	return q
}

func CompoundDivide(inflated, seed *big.Int) uint64 {
	num := new(big.Int).Lsh(inflated, 48)
	z := num.Div(num, seed)
	z.Sub(z, Q48)
	if z.Cmp(Q48) >= 0 {
		return Q48.Uint64()
	}
	return z.Uint64()
}

func CompoundPrice(price *big.Int, growth uint64, shiftUp bool) *big.Int {
	multFactor := new(big.Int).SetUint64(growth)
	multFactor.Add(multFactor, Q48)
	if shiftUp {
		z := new(big.Int).Mul(price, multFactor)
		z.Rsh(z, 48)
		return z.Add(z, bignum.One)
	}
	z := new(big.Int).Lsh(price, 48)
	return z.Div(z, multFactor)
}

func InflateLiqSeed(seed *big.Int, growth uint64) *big.Int {
	multFactor := new(big.Int).SetUint64(growth)
	multFactor.Add(multFactor, Q48)
	inflated := multFactor.Mul(seed, multFactor)
	inflated.Rsh(inflated, 48)
	if inflated.Cmp(mask128) > 0 {
		return new(big.Int).Set(mask128)
	}
	return inflated
}
