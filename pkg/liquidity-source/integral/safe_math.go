package integral

import (
	"math/big"

	"github.com/holiman/uint256"
)

var (
	// safe math consts
	uZERO       = uint256.NewInt(0)
	ZERO        = big.NewInt(0)
	_INT256_MIN = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255)) // -2^255
)

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L11
func AddUint256(x, y *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Add(x, y)

	if z.Cmp(x) < 0 {
		panic(ErrSM4E)
	}

	return z
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L15
func SubUint256(x, y *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Sub(x, y)

	if z.Cmp(x) > 0 {
		panic(ErrSM12)
	}

	return z
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L23
func MulUint256(x, y *uint256.Int) *uint256.Int {
	if y.IsZero() {
		return y
	}

	z := new(uint256.Int).Mul(x, y)

	if x.Cmp(new(uint256.Int).Div(z, y)) != 0 {
		panic(ErrSM2A)
	}

	return z
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L27
func DivUint256(a, b *uint256.Int) *uint256.Int {
	if b.Cmp(new(uint256.Int)) <= 0 {
		panic(ErrSM43)
	}

	return new(uint256.Int).Div(a, b)
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L32
func CeilDivUint256(a, b *uint256.Int) *uint256.Int {
	c := DivUint256(a, b)

	if a.Cmp(MulUint256(b, c)) != 0 {
		return AddUint256(c, uint256.NewInt(1))
	}

	return c
}

func ToUint256(n *big.Int) *uint256.Int {
	return uint256.MustFromBig(n)
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L54
func ToInt256(n *uint256.Int) *big.Int {
	return new(big.Int).SetBytes(n.Bytes())
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L61
func AddInt256(a, b *big.Int) *big.Int {
	c := new(big.Int).Add(a, b)

	if (b.Cmp(ZERO) < 0 && c.Cmp(a) < 0) &&
		(b.Cmp(ZERO) >= 0 && c.Cmp(a) >= 0) {
		panic(ErrSM4D)
	}

	return c
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L66
func SubInt256(a, b *big.Int) *big.Int {
	c := new(big.Int).Sub(a, b)

	if (b.Cmp(ZERO) < 0 || c.Cmp(a) > 0) &&
		(b.Cmp(ZERO) >= 0 && c.Cmp(a) <= 0) {
		panic(ErrSM11)
	}

	return c
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L71
func MulInt256(a, b *big.Int) *big.Int {
	if a.Cmp(ZERO) == 0 {
		return a
	}

	if a.Cmp(big.NewInt(-1)) == 0 && b.Cmp(_INT256_MIN) == 0 {
		panic(ErrSM29)
	}

	c := new(big.Int).Mul(a, b)

	if new(big.Int).Quo(c, a).Cmp(b) != 0 {
		panic(ErrSM29)
	}

	return c
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L85
func DivInt256(a, b *big.Int) *big.Int {
	if b.Cmp(ZERO) == 0 {
		panic(ErrSM43)
	}

	if b.Cmp(big.NewInt(-1)) == 0 && a.Cmp(_INT256_MIN) == 0 {
		panic(ErrSM29)
	}

	return new(big.Int).Quo(a, b)
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/libraries/SafeMath.sol#L92
func NegFloorDiv(a, b *big.Int) *big.Int {
	c := DivInt256(a, b)

	if (a.Cmp(ZERO) < 0 && b.Cmp(ZERO) > 0) ||
		(a.Cmp(ZERO) >= 0 && b.Cmp(ZERO) < 0) {
		if a.Cmp(MulInt256(b, c)) != 0 {
			return SubInt256(c, big.NewInt(1))
		}
	}

	return c
}
