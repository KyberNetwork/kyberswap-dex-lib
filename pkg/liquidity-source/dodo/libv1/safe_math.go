package libv1

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// SafeMul https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/SafeMath.sol#L19
func SafeMul(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(number.Zero) == 0 {
		return number.Zero
	}

	c := new(uint256.Int).Mul(a, b)
	if new(uint256.Int).Div(c, a).Cmp(b) != 0 {
		panic(ErrMulError)
	}

	return c
}

// SafeDiv https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/SafeMath.sol#L30
func SafeDiv(a, b *uint256.Int) *uint256.Int {
	if b.Cmp(number.Zero) <= 0 {
		panic(ErrDividingError)
	}

	return new(uint256.Int).Div(a, b)
}

// SafeDivCeil https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/SafeMath.sol#L35
func SafeDivCeil(a, b *uint256.Int) *uint256.Int {
	quotient := new(uint256.Int).Div(a, b)
	remainder := new(uint256.Int).Sub(a, new(uint256.Int).Mul(quotient, b))
	if remainder.Cmp(number.Zero) > 0 {
		return new(uint256.Int).Add(quotient, number.Number_1)
	}

	return quotient
}

// SafeSub https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/SafeMath.sol#L45
func SafeSub(a, b *uint256.Int) *uint256.Int {
	if b.Cmp(a) > 0 {
		panic(ErrSubError)
	}

	return new(uint256.Int).Sub(a, b)
}

// SafeAdd https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/SafeMath.sol#L50
func SafeAdd(a, b *uint256.Int) *uint256.Int {
	c := new(uint256.Int).Add(a, b)
	if c.Cmp(a) < 0 {
		panic(ErrAddError)
	}

	return c
}

// SafeSqrt https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/SafeMath.sol#L56
func SafeSqrt(x *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Add(new(uint256.Int).Div(x, number.Number_2), number.Number_1)
	y := x
	for z.Cmp(y) < 0 {
		y = z
		z = new(uint256.Int).Div(new(uint256.Int).Add(new(uint256.Int).Div(x, z), z), number.Number_2)
	}

	return y
}
