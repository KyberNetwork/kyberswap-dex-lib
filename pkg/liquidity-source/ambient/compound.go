package ambient

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func ApproxSqrtCompound(x64 uint64) uint64 {
	// Saturate at Q48-1. The on-chain contract reverts when x >= Q48 but in
	// the quoting path we'd rather produce a degraded result than crash.
	if x64 >= Q48.Uint64() {
		x64 = Q48.Uint64() - 1
	}
	x := new(big.Int).SetUint64(x64)
	xSq := new(big.Int).Mul(x, x)
	xSq.Rsh(xSq, 48)
	linear := new(big.Int).Rsh(x, 1)
	quad := new(big.Int).Rsh(xSq, 3)
	result := new(big.Int).Sub(linear, quad)
	return result.Uint64()
}

func CompoundStack(x, y uint64) uint64 {
	one := Q48.Uint64()
	num := new(big.Int).Mul(
		new(big.Int).SetUint64(one+x),
		new(big.Int).SetUint64(one+y),
	)
	term := new(big.Int).Rsh(num, 48)
	z := term.Sub(term, Q48)
	if z.Cmp(new(big.Int).SetUint64(^uint64(0))) >= 0 {
		return ^uint64(0)
	}
	return z.Uint64()
}

func CompoundShrink(val, deflator uint64) uint64 {
	one := Q48.Uint64()
	multFactor := new(big.Int).SetUint64(one + deflator)
	num := new(big.Int).Lsh(new(big.Int).SetUint64(val), 48)
	z := num.Div(num, multFactor)
	return z.Uint64()
}

func CompoundDivide(inflated, seed *big.Int) uint64 {
	num := new(big.Int).Lsh(new(big.Int).Set(inflated), 48)
	z := new(big.Int).Div(num, seed)
	z.Sub(z, Q48)
	if z.Cmp(Q48) >= 0 {
		return Q48.Uint64()
	}
	return z.Uint64()
}

func CompoundPrice(price *big.Int, growth uint64, shiftUp bool) *big.Int {
	multFactor := new(big.Int).Add(Q48, new(big.Int).SetUint64(growth))
	if shiftUp {
		num := new(big.Int).Mul(price, multFactor)
		z := new(big.Int).Rsh(num, 48)
		z.Add(z, bignum.One)
		return z
	}
	num := new(big.Int).Lsh(new(big.Int).Set(price), 48)
	return num.Div(num, multFactor)
}

func InflateLiqSeed(seed *big.Int, growth uint64) *big.Int {
	multFactor := new(big.Int).Add(Q48, new(big.Int).SetUint64(growth))
	num := new(big.Int).Mul(seed, multFactor)
	inflated := new(big.Int).Rsh(num, 48)
	if inflated.Cmp(mask128) > 0 {
		return new(big.Int).Set(mask128)
	}
	return inflated
}
