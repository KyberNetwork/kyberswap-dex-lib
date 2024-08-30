package integral

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	_INT256_MIN = new(int256.Int).SetUint64(0) // -2^255
	ErrT027     = errors.New("T027")
	ErrSM43     = errors.New("SM43")
	ErrSM4E     = errors.New("SM4E")
	ErrSM12     = errors.New("SM12")
	ErrSM2A     = errors.New("SM2A")
	ErrSM4D     = errors.New("SM4D")
	ErrSM11     = errors.New("SM11")
	ErrSM29     = errors.New("SM29")
	ErrSM42     = errors.New("SM42")
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

func ToUint32(n *int256.Int) uint32 {
	return 0
}

func ToUint64(n *int256.Int) uint64 {
	return 0
}

func ToUint112(n *int256.Int) *uint256.Int {
	return nil
}

func ToInt256(unsigned *uint256.Int) *uint256.Int {
	return nil
}

func AddInt256(a, b *int256.Int) *int256.Int {
	c := new(int256.Int).Add(a, b)

	if (b.Cmp(new(int256.Int)) < 0 && c.Cmp(a) < 0) ||
		(b.Cmp(new(int256.Int)) >= 0 && c.Cmp(a) >= 0) {
		panic(ErrSM4D)
	}

	return c
}

func SubInt256(a, b *int256.Int) *int256.Int {
	c := new(int256.Int).Sub(a, b)

	if (b.Cmp(new(int256.Int)) < 0 && c.Cmp(a) > 0) ||
		(b.Cmp(new(int256.Int)) >= 0 && c.Cmp(a) <= 0) {
		panic(ErrSM11)
	}

	return c
}

func MulInt256(a, b *int256.Int) *int256.Int {
	if a.IsZero() {
		return a
	}

	if a.Cmp(int256.NewInt(-1)) == 0 && b.Cmp(_INT256_MIN) == 0 {
		panic(ErrSM29)
	}

	c := new(int256.Int).Mul(a, b)

	if new(int256.Int).Quo(c, a).Cmp(b) != 0 {
		panic(ErrSM29)
	}

	return c
}

func DivInt256(a, b *int256.Int) *int256.Int {
	if a.IsZero() {
		panic(ErrSM43)
	}

	if b.Cmp(int256.NewInt(-1)) == 0 && a.Cmp(_INT256_MIN) == 0 {
		panic(ErrSM29)
	}

	return new(int256.Int).Quo(a, b)
}

func NegFloorDiv(a, b *int256.Int) *int256.Int {
	c := DivInt256(a, b)

	if (a.Cmp(new(int256.Int)) < 0 && b.Cmp(new(int256.Int)) > 0) ||
		(a.Cmp(new(int256.Int)) >= 0 && b.Cmp(new(int256.Int)) < 0) {
		if a.Cmp(MulInt256(b, c)) != 0 {
			return SubInt256(c, int256.NewInt(1))
		}
	}

	return c
}
