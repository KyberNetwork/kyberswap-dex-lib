package integral

import (
	"math/big"

	"github.com/holiman/uint256"
)

func AddUint256(x, y *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Add(x, y)

	if z.Cmp(x) < 0 {
		panic(ErrSM4E)
	}

	return z
}

func SubUint256(x, y *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Sub(x, y)

	if z.Cmp(x) > 0 {
		panic(ErrSM12)
	}

	return z
}

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

func DivUint256(a, b *uint256.Int) *uint256.Int {
	if b.Cmp(new(uint256.Int)) <= 0 {
		panic(ErrSM43)
	}

	return new(uint256.Int).Div(a, b)
}

func CeilDivUint256(a, b *uint256.Int) *uint256.Int {
	c := DivUint256(a, b)

	if a.Cmp(MulUint256(b, c)) != 0 {
		return AddUint256(c, uint256.NewInt(1))
	}

	return c
}

func ToUint256(n *big.Int) *uint256.Int {
	return new(uint256.Int).SetBytes(n.Bytes())
}

func ToInt256(n *uint256.Int) *big.Int {
	return new(big.Int).SetBytes(n.Bytes())
}

func AddInt256(a, b *big.Int) *big.Int {
	c := new(big.Int).Add(a, b)

	if (b.Cmp(ZERO) < 0 && c.Cmp(a) < 0) &&
		(b.Cmp(ZERO) >= 0 && c.Cmp(a) >= 0) {
		panic(ErrSM4D)
	}

	return c
}

func SubInt256(a, b *big.Int) *big.Int {
	c := new(big.Int).Sub(a, b)

	if (b.Cmp(ZERO) < 0 || c.Cmp(a) > 0) &&
		(b.Cmp(ZERO) >= 0 && c.Cmp(a) <= 0) {
		panic(ErrSM11)
	}

	return c
}

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

func DivInt256(a, b *big.Int) *big.Int {
	if b.Cmp(ZERO) == 0 {
		panic(ErrSM43)
	}

	if b.Cmp(big.NewInt(-1)) == 0 && a.Cmp(_INT256_MIN) == 0 {
		panic(ErrSM29)
	}

	return new(big.Int).Quo(a, b)
}

func NegFloorDiv(a, b *big.Int) *big.Int {
	c := DivInt256(a, b)

	if (a.Cmp(ZERO) >= 0 || b.Cmp(ZERO) <= 0) &&
		(a.Cmp(ZERO) < 0 || b.Cmp(ZERO) >= 0) {
		if a.Cmp(MulInt256(b, c)) != 0 {
			return SubInt256(c, big.NewInt(1))
		}
	}

	return c
}
