package pufeth

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

var (
	ErrOverflow = errors.New("overflow")
)

type (
	math struct {
	}

	Uint512 struct {
		Hi *uint256.Int
		Lo *uint256.Int
	}
)

var (
	Math *math

	// 2^256 - 1
	maxUint256 = new(uint256.Int).Sub(
		new(uint256.Int).Lsh(number.Number_1, 256),
		number.Number_1,
	)
)

func init() {
	Math = &math{}
}

func (m *math) MulDivF(x, y, z *uint256.Int) (*uint256.Int, error) {
	xy := m.mul512(x, y)
	if xy.Hi.IsZero() {
		return new(uint256.Int).Div(xy.Lo, z), nil
	}

	if xy.Hi.Cmp(z) >= 0 {
		return nil, ErrOverflow
	}

	_m := new(uint256.Int).MulMod(x, y, z)
	n := m._sub512(xy, _m)

	if n.Hi.IsZero() {
		return new(uint256.Int).Div(n.Lo, z), nil
	}

	p := new(uint256.Int).And(
		m._unsafeSub(number.Zero, z),
		z,
	)
	q := m._div512(n, p)
	r := m._inv256(new(uint256.Int).Div(z, p))
	return m._unsafeMul(q, r), nil
}

func (m *math) _inv256(d *uint256.Int) *uint256.Int {
	x := number.Number_1
	for i := 0; i < 8; i++ {
		x = m._unsafeMul(x, m._unsafeSub(number.Number_2, m._unsafeMul(x, d)))
	}
	return x
}

func (m *math) _div512(x *Uint512, pow2n *uint256.Int) *uint256.Int {
	pow2nInv := m._unsafeAdd(
		new(uint256.Int).Div(
			m._unsafeSub(number.Zero, pow2n),
			pow2n,
		),
		number.Number_1,
	)
	return new(uint256.Int).Or(
		m._unsafeMul(x.Hi, pow2nInv),
		new(uint256.Int).Div(x.Lo, pow2n),
	)
}

func (m *math) _sub512(x *Uint512, y *uint256.Int) *Uint512 {
	if x.Lo.Cmp(y) >= 0 {
		return &Uint512{
			Hi: x.Hi,
			Lo: new(uint256.Int).Sub(x.Lo, y),
		}
	}
	return &Uint512{
		Hi: new(uint256.Int).Sub(x.Hi, number.Number_1),
		Lo: m._unsafeSub(x.Lo, y),
	}
}

func (m *math) mul512(x, y *uint256.Int) *Uint512 {
	p := m._mulModMax(x, y)
	q := m._unsafeMul(x, y)
	if p.Cmp(q) >= 0 {
		return &Uint512{
			Hi: new(uint256.Int).Sub(p, q),
			Lo: q,
		}
	}
	return &Uint512{
		Hi: new(uint256.Int).Sub(m._unsafeSub(p, q), number.Number_1),
		Lo: q,
	}
}

func (m *math) _unsafeAdd(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Add(x, y)
}

func (m *math) _unsafeSub(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Sub(x, y)
}

func (m *math) _unsafeMul(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Mul(x, y)
}

func (m *math) _mulModMax(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).MulMod(x, y, maxUint256)
}
