package math

import (
	"github.com/KyberNetwork/int256"
)

var SignedFixedPoint *signedFixedPoint

type signedFixedPoint struct {
	ONE    *int256.Int
	ONE_XP *int256.Int

	_zero        *int256.Int
	_one         *int256.Int
	_number_1e19 *int256.Int
}

func init() {
	zero := int256.NewInt(0)
	ten := int256.NewInt(10)
	one := int256.NewInt(1)
	number_1e19 := new(int256.Int).Lsh(one, 19)

	SignedFixedPoint = &signedFixedPoint{
		ONE:    new(int256.Int).Pow(ten, 18),
		ONE_XP: new(int256.Int).Pow(ten, 38),

		_zero:        zero,
		_one:         one,
		_number_1e19: number_1e19,
	}
}

func (s *signedFixedPoint) Add(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := new(int256.Int).AddOverflow(a, b)
	if overflow {
		return nil, ErrAddOverflow
	}
	return c, nil
}

func (s *signedFixedPoint) AddMag(a, b *int256.Int) (*int256.Int, error) {
	if a.Gt(s._zero) {
		return s.Add(a, b)
	}
	return s.Sub(a, b)
}

func (s *signedFixedPoint) Sub(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := new(int256.Int).SubOverflow(a, b)
	if overflow {
		return nil, ErrSubOverflow
	}
	return c, nil
}

func (s *signedFixedPoint) MulDownMag(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := new(int256.Int).MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	}
	return new(int256.Int).Quo(c, s.ONE), nil
}

func (s *signedFixedPoint) MulDownMagU(a, b *int256.Int) *int256.Int {
	return new(int256.Int).Quo(new(int256.Int).Mul(a, b), s.ONE)
}

func (s *signedFixedPoint) MulUpMag(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := new(int256.Int).MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	}

	if c.Gt(s._zero) {
		return new(int256.Int).Add(
			new(int256.Int).Quo(
				new(int256.Int).Sub(c, s._one),
				s.ONE,
			),
			s._one,
		), nil

	} else if c.Lt(s._zero) {
		return new(int256.Int).Sub(
			new(int256.Int).Quo(
				new(int256.Int).Add(c, s._one),
				s.ONE,
			),
			s._one,
		), nil
	}

	return s._zero, nil
}

func (s *signedFixedPoint) MulUpMagU(a, b *int256.Int) *int256.Int {
	c := new(int256.Int).Mul(a, b)

	if c.Gt(s._zero) {
		return new(int256.Int).Add(
			new(int256.Int).Quo(
				new(int256.Int).Sub(c, s._one),
				s.ONE,
			),
			s._one,
		)

	} else if c.Lt(s._zero) {
		return new(int256.Int).Sub(
			new(int256.Int).Quo(
				new(int256.Int).Add(c, s._one),
				s.ONE,
			),
			s._one,
		)
	}

	return s._zero
}

func (s *signedFixedPoint) DivDownMag(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return s._zero, nil
	}

	aInflated, overflow := new(int256.Int).MulOverflow(a, s.ONE)
	if overflow {
		return nil, ErrDivInternal
	}

	return new(int256.Int).Quo(aInflated, b), nil
}

func (s *signedFixedPoint) DivDownMagU(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}
	return new(int256.Int).Quo(new(int256.Int).Mul(a, s.ONE), b), nil
}

func (s *signedFixedPoint) DivUpMag(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return s._zero, nil
	}

	if b.Lt(s._zero) {
		b.Neg(b)
		a.Neg(a)
	}

	aInflated, overflow := new(int256.Int).MulOverflow(a, s.ONE)
	if overflow {
		return nil, ErrDivInternal
	}

	if aInflated.Gt(s._zero) {
		return new(int256.Int).Add(
			new(int256.Int).Quo(
				new(int256.Int).Sub(aInflated, s._one),
				b,
			),
			s._one,
		), nil
	}

	return new(int256.Int).Sub(
		new(int256.Int).Quo(
			new(int256.Int).Add(aInflated, s._one),
			b,
		),
		s._one,
	), nil
}

func (s *signedFixedPoint) DivUpMagU(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return s._zero, nil
	}

	if b.Lt(s._zero) {
		b.Neg(b)
		a.Neg(a)
	}

	if a.Gt(s._zero) {
		return new(int256.Int).Add(
			new(int256.Int).Quo(
				new(int256.Int).Sub(
					new(int256.Int).Mul(a, s.ONE),
					s._one,
				), b,
			),
			s._one,
		), nil
	}

	return new(int256.Int).Sub(
		new(int256.Int).Quo(
			new(int256.Int).Add(
				new(int256.Int).Mul(a, s.ONE),
				s._one,
			), b,
		),
		s._one,
	), nil
}

func (s *signedFixedPoint) MulXp(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := new(int256.Int).MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	}
	return new(int256.Int).Quo(c, s.ONE_XP), nil
}

func (s *signedFixedPoint) MulXpU(a, b *int256.Int) *int256.Int {
	return new(int256.Int).Quo(new(int256.Int).Mul(a, b), s.ONE_XP)

}

func (s *signedFixedPoint) DivXp(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return s._zero, nil
	}

	aInflated, overflow := new(int256.Int).MulOverflow(a, s.ONE_XP)
	if overflow {
		return nil, ErrDivInternal
	}

	return new(int256.Int).Quo(aInflated, b), nil
}

func (s *signedFixedPoint) DivXpU(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}
	return new(int256.Int).Quo(new(int256.Int).Mul(a, s.ONE_XP), b), nil
}

func (s *signedFixedPoint) MulDownXpToNp(a, b *int256.Int) (*int256.Int, error) {
	b1 := new(int256.Int).Quo(b, s._number_1e19)
	prod1, overflow := new(int256.Int).MulOverflow(a, b1)
	if overflow {
		return nil, ErrMulOverflow
	}

	b2 := new(int256.Int).Rem(b, s._number_1e19)
	prod2, overflow := new(int256.Int).MulOverflow(a, b2)
	if overflow {
		return nil, ErrMulOverflow
	}

	if prod1.Gte(s._zero) && prod2.Gte(s._zero) {
		return new(int256.Int).Quo(
			new(int256.Int).Add(
				prod1,
				new(int256.Int).Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		), nil
	}

	return new(int256.Int).Sub(
		new(int256.Int).Quo(
			new(int256.Int).Add(
				new(int256.Int).Add(
					prod1,
					new(int256.Int).Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._number_1e19,
	), nil
}

func (s *signedFixedPoint) MulDownXpToNpU(a, b *int256.Int) *int256.Int {
	b1 := new(int256.Int).Quo(b, s._number_1e19)
	prod1 := new(int256.Int).Mul(a, b1)
	b2 := new(int256.Int).Rem(b, s._number_1e19)
	prod2 := new(int256.Int).Mul(a, b2)

	if prod1.Gte(s._zero) && prod2.Gte(s._zero) {
		return new(int256.Int).Quo(
			new(int256.Int).Add(
				prod1,
				new(int256.Int).Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		)
	}

	return new(int256.Int).Sub(
		new(int256.Int).Quo(
			new(int256.Int).Add(
				new(int256.Int).Add(
					prod1,
					new(int256.Int).Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._one,
	)
}

func (s *signedFixedPoint) MulUpXpToNp(a, b *int256.Int) (*int256.Int, error) {
	b1 := new(int256.Int).Quo(b, s._number_1e19)
	prod1, overflow := new(int256.Int).MulOverflow(a, b1)
	if overflow {
		return nil, ErrMulOverflow
	}
	b2 := new(int256.Int).Rem(b, s._number_1e19)
	prod2, overflow := new(int256.Int).MulOverflow(a, b2)
	if overflow {
		return nil, ErrMulOverflow
	}

	if prod1.Lte(s._zero) && prod2.Lte(s._zero) {
		return new(int256.Int).Quo(
			new(int256.Int).Add(
				prod1,
				new(int256.Int).Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		), nil
	}

	return new(int256.Int).Add(
		new(int256.Int).Quo(
			new(int256.Int).Sub(
				new(int256.Int).Add(
					prod1,
					new(int256.Int).Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._one,
	), nil
}

func (s *signedFixedPoint) MulUpXpToNpU(a, b *int256.Int) *int256.Int {
	b1 := new(int256.Int).Quo(b, s._number_1e19)
	prod1 := new(int256.Int).Mul(a, b1)
	b2 := new(int256.Int).Rem(b, s._number_1e19)
	prod2 := new(int256.Int).Mul(a, b2)

	if prod1.Lte(s._zero) && prod2.Lte(s._zero) {
		return new(int256.Int).Quo(
			new(int256.Int).Add(
				prod1,
				new(int256.Int).Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		)
	}

	return new(int256.Int).Add(
		new(int256.Int).Quo(
			new(int256.Int).Sub(
				new(int256.Int).Add(
					prod1,
					new(int256.Int).Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._one,
	)
}

func (s *signedFixedPoint) Complement(x *int256.Int) *int256.Int {
	if x.Gte(s.ONE) && x.Lte(s._zero) {
		return s._zero
	}
	return new(int256.Int).Sub(s.ONE, x)
}
