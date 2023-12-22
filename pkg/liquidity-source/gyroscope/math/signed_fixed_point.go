package math

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

var SignedFixedPoint *signedFixedPoint

type signedFixedPoint struct {
	ONE    *big.Int
	ONE_XP *big.Int
}

func init() {
	SignedFixedPoint = &signedFixedPoint{
		ONE:    integer.TenPow(18),
		ONE_XP: integer.TenPow(38),
	}
}

func (s *signedFixedPoint) Add(a, b *big.Int) (*big.Int, error) {
	c := new(big.Int).Add(a, b)
	if c.Cmp(maxI256) > 0 || c.Cmp(minI256) < 0 {
		return nil, ErrAddOverflow
	}
	return c, nil
}

func (s *signedFixedPoint) AddMag(a, b *big.Int) (*big.Int, error) {
	if a.Cmp(integer.Zero()) > 0 {
		return s.Add(a, b)
	}
	return s.Sub(a, b)
}

func (s *signedFixedPoint) Sub(a, b *big.Int) (*big.Int, error) {
	c := new(big.Int).Sub(a, b)
	if c.Cmp(maxI256) > 0 || c.Cmp(minI256) < 0 {
		return nil, ErrSubOverflow
	}
	return c, nil
}

func (s *signedFixedPoint) MulDownMag(a, b *big.Int) (*big.Int, error) {
	c := new(big.Int).Mul(a, b)
	if c.Cmp(maxI256) > 0 || c.Cmp(minI256) < 0 {
		return nil, ErrMulOverflow
	}
	return new(big.Int).Quo(c, s.ONE), nil
}

func (s *signedFixedPoint) MulDownMagU(a, b *big.Int) *big.Int {
	return new(big.Int).Quo(new(big.Int).Mul(a, b), s.ONE)
}

func (s *signedFixedPoint) MulUpMag(a, b *big.Int) (*big.Int, error) {
	c := new(big.Int).Mul(a, b)
	if c.Cmp(maxI256) > 0 || c.Cmp(minI256) < 0 {
		return nil, ErrMulOverflow
	}

	if c.Cmp(integer.Zero()) > 0 {
		return new(big.Int).Add(
			new(big.Int).Quo(
				new(big.Int).Sub(c, integer.One()),
				s.ONE,
			),
			integer.One(),
		), nil

	} else if c.Cmp(integer.Zero()) < 0 {
		return new(big.Int).Sub(
			new(big.Int).Quo(
				new(big.Int).Add(c, integer.One()),
				s.ONE,
			),
			integer.One(),
		), nil
	}

	return integer.Zero(), nil
}

func (s *signedFixedPoint) MulUpMagU(a, b *big.Int) *big.Int {
	c := new(big.Int).Mul(a, b)

	if c.Cmp(integer.Zero()) > 0 {
		return new(big.Int).Add(
			new(big.Int).Quo(
				new(big.Int).Sub(c, integer.One()),
				s.ONE,
			),
			integer.One(),
		)

	} else if c.Cmp(integer.Zero()) < 0 {
		return new(big.Int).Sub(
			new(big.Int).Quo(
				new(big.Int).Add(c, integer.One()),
				s.ONE,
			),
			integer.One(),
		)
	}

	return integer.Zero()
}

func (s *signedFixedPoint) DivDownMag(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(integer.Zero()) == 0 {
		return nil, ErrZeroDivision
	}

	if a.Cmp(integer.Zero()) == 0 {
		return integer.Zero(), nil
	}

	aInflated := new(big.Int).Mul(a, s.ONE)
	if aInflated.Cmp(maxI256) > 0 || aInflated.Cmp(minI256) < 0 {
		return nil, ErrDivInternal
	}

	return new(big.Int).Quo(aInflated, b), nil
}

func (s *signedFixedPoint) DivDownMagU(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(integer.Zero()) == 0 {
		return nil, ErrZeroDivision
	}
	return new(big.Int).Quo(new(big.Int).Mul(a, s.ONE), b), nil
}

func (s *signedFixedPoint) DivUpMag(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(integer.Zero()) == 0 {
		return nil, ErrZeroDivision
	}

	if a.Cmp(integer.Zero()) == 0 {
		return integer.Zero(), nil
	}

	if b.Cmp(integer.Zero()) < 0 {
		b = new(big.Int).Neg(b)
		a = new(big.Int).Neg(a)
	}

	aInflated := new(big.Int).Mul(a, s.ONE)
	if aInflated.Cmp(maxI256) > 0 || aInflated.Cmp(minI256) < 0 {
		return nil, ErrDivInternal
	}

	if aInflated.Cmp(integer.Zero()) > 0 {
		return new(big.Int).Add(
			new(big.Int).Quo(
				new(big.Int).Sub(aInflated, integer.One()),
				b,
			),
			integer.One(),
		), nil
	}

	return new(big.Int).Sub(
		new(big.Int).Quo(
			new(big.Int).Add(aInflated, integer.One()),
			b,
		),
		integer.One(),
	), nil
}

func (s *signedFixedPoint) DivUpMagU(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(integer.Zero()) == 0 {
		return nil, ErrZeroDivision
	}

	if a.Cmp(integer.Zero()) == 0 {
		return integer.Zero(), nil
	}

	if b.Cmp(integer.Zero()) < 0 {
		b = new(big.Int).Neg(b)
		a = new(big.Int).Neg(a)
	}

	if a.Cmp(integer.Zero()) > 0 {
		return new(big.Int).Add(
			new(big.Int).Quo(
				new(big.Int).Sub(
					new(big.Int).Mul(a, s.ONE),
					integer.One(),
				), b,
			),
			integer.One(),
		), nil
	}

	return new(big.Int).Sub(
		new(big.Int).Quo(
			new(big.Int).Add(
				new(big.Int).Mul(a, s.ONE),
				integer.One(),
			), b,
		),
		integer.One(),
	), nil
}

func (s *signedFixedPoint) MulXp(a, b *big.Int) (*big.Int, error) {
	c := new(big.Int).Mul(a, b)
	if c.Cmp(maxI256) > 0 || c.Cmp(minI256) < 0 {
		return nil, ErrMulOverflow
	}
	return new(big.Int).Quo(c, s.ONE_XP), nil
}

func (s *signedFixedPoint) MulXpU(a, b *big.Int) *big.Int {
	return new(big.Int).Quo(new(big.Int).Mul(a, b), s.ONE_XP)
}

func (s *signedFixedPoint) DivXp(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(integer.Zero()) == 0 {
		return nil, ErrZeroDivision
	}

	if a.Cmp(integer.Zero()) == 0 {
		return integer.Zero(), nil
	}

	aInflated := new(big.Int).Mul(a, s.ONE_XP)
	if aInflated.Cmp(maxI256) > 0 || aInflated.Cmp(minI256) < 0 {
		return nil, ErrDivInternal
	}

	return new(big.Int).Quo(aInflated, b), nil
}

func (s *signedFixedPoint) DivXpU(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(integer.Zero()) == 0 {
		return nil, ErrZeroDivision
	}
	return new(big.Int).Quo(new(big.Int).Mul(a, s.ONE_XP), b), nil
}

func (s *signedFixedPoint) MulDownXpToNp(a, b *big.Int) (*big.Int, error) {
	b1 := new(big.Int).Quo(b, bignumber1e19)
	prod1 := new(big.Int).Mul(a, b1)
	if prod1.Cmp(maxI256) > 0 || prod1.Cmp(minI256) < 0 {
		return nil, ErrMulOverflow
	}
	b2 := new(big.Int).Mod(b, bignumber1e19)
	prod2 := new(big.Int).Mul(a, b2)
	if prod2.Cmp(maxI256) > 0 || prod2.Cmp(minI256) < 0 {
		return nil, ErrMulOverflow
	}

	if prod1.Cmp(integer.Zero()) >= 0 && prod2.Cmp(integer.Zero()) >= 0 {
		return new(big.Int).Quo(
			new(big.Int).Add(
				prod1,
				new(big.Int).Quo(prod2, bignumber1e19),
			),
			bignumber1e19,
		), nil
	}

	return new(big.Int).Sub(
		new(big.Int).Quo(
			new(big.Int).Add(
				new(big.Int).Add(
					prod1,
					new(big.Int).Quo(prod2, bignumber1e19),
				),
				integer.One(),
			),
			bignumber1e19,
		),
		integer.One(),
	), nil
}

func (s *signedFixedPoint) MulDownXpToNpU(a, b *big.Int) *big.Int {
	b1 := new(big.Int).Quo(b, bignumber1e19)
	prod1 := new(big.Int).Mul(a, b1)
	b2 := new(big.Int).Mod(b, bignumber1e19)
	prod2 := new(big.Int).Mul(a, b2)

	if prod1.Cmp(integer.Zero()) >= 0 && prod2.Cmp(integer.Zero()) >= 0 {
		return new(big.Int).Quo(
			new(big.Int).Add(
				prod1,
				new(big.Int).Quo(prod2, bignumber1e19),
			),
			bignumber1e19,
		)
	}

	return new(big.Int).Sub(
		new(big.Int).Quo(
			new(big.Int).Add(
				new(big.Int).Add(
					prod1,
					new(big.Int).Quo(prod2, bignumber1e19),
				),
				integer.One(),
			),
			bignumber1e19,
		),
		integer.One(),
	)
}

func (s *signedFixedPoint) MulUpXpToNp(a, b *big.Int) (*big.Int, error) {
	b1 := new(big.Int).Quo(b, bignumber1e19)
	prod1 := new(big.Int).Mul(a, b1)
	if prod1.Cmp(maxI256) > 0 || prod1.Cmp(minI256) < 0 {
		return nil, ErrMulOverflow
	}
	b2 := new(big.Int).Mod(b, bignumber1e19)
	prod2 := new(big.Int).Mul(a, b2)
	if prod2.Cmp(maxI256) > 0 || prod2.Cmp(minI256) < 0 {
		return nil, ErrMulOverflow
	}

	if prod1.Cmp(integer.Zero()) <= 0 && prod2.Cmp(integer.Zero()) <= 0 {
		return new(big.Int).Quo(
			new(big.Int).Add(
				prod1,
				new(big.Int).Quo(prod2, bignumber1e19),
			),
			bignumber1e19,
		), nil
	}

	return new(big.Int).Add(
		new(big.Int).Quo(
			new(big.Int).Sub(
				new(big.Int).Add(
					prod1,
					new(big.Int).Quo(prod2, bignumber1e19),
				),
				integer.One(),
			),
			bignumber1e19,
		),
		integer.One(),
	), nil
}

func (s *signedFixedPoint) MulUpXpToNpU(a, b *big.Int) *big.Int {
	b1 := new(big.Int).Quo(b, bignumber1e19)
	prod1 := new(big.Int).Mul(a, b1)
	b2 := new(big.Int).Mod(b, bignumber1e19)
	prod2 := new(big.Int).Mul(a, b2)

	if prod1.Cmp(integer.Zero()) <= 0 && prod2.Cmp(integer.Zero()) <= 0 {
		return new(big.Int).Quo(
			new(big.Int).Add(
				prod1,
				new(big.Int).Quo(prod2, bignumber1e19),
			),
			bignumber1e19,
		)
	}

	return new(big.Int).Add(
		new(big.Int).Quo(
			new(big.Int).Sub(
				new(big.Int).Add(
					prod1,
					new(big.Int).Quo(prod2, bignumber1e19),
				),
				integer.One(),
			),
			bignumber1e19,
		),
		integer.One(),
	)
}

func (s *signedFixedPoint) Complement(x *big.Int) *big.Int {
	if x.Cmp(s.ONE) >= 0 && x.Cmp(integer.Zero()) <= 0 {
		return integer.Zero()
	}
	return new(big.Int).Sub(s.ONE, x)
}
