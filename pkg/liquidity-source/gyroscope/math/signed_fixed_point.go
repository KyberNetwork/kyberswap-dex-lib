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
	number_1e19 := new(int256.Int).Pow(ten, 19)

	SignedFixedPoint = &signedFixedPoint{
		ONE:    new(int256.Int).Pow(ten, 18),
		ONE_XP: new(int256.Int).Pow(ten, 38),

		_zero:        zero,
		_one:         one,
		_number_1e19: number_1e19,
	}
}

func (s *signedFixedPoint) Add(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := a.AddOverflow(a, b)
	if overflow {
		return nil, ErrAddOverflow
	}
	return c, nil
}

func (s *signedFixedPoint) AddMag(a, b *int256.Int) (*int256.Int, error) {
	if a.IsPositive() {
		return s.Add(a, b)
	}
	return s.Sub(a, b)
}

func (s *signedFixedPoint) Sub(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := a.SubOverflow(a, b)
	if overflow {
		return nil, ErrSubOverflow
	}
	return c, nil
}

func (s *signedFixedPoint) MulDownMag(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := a.MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	}
	return c.Quo(c, s.ONE), nil
}

func (s *signedFixedPoint) MulDownMagU(a, b *int256.Int) *int256.Int {
	return a.Quo(a.Mul(a, b), s.ONE)
}

func (s *signedFixedPoint) MulUpMag(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := a.MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	} else if c.IsPositive() {
		return c.Add(
			c.Quo(
				c.Sub(c, s._one),
				s.ONE,
			),
			s._one,
		), nil
	} else if c.IsNegative() {
		return c.Sub(
			c.Quo(
				c.Add(c, s._one),
				s.ONE,
			),
			s._one,
		), nil
	}

	return c.Clear(), nil
}

func (s *signedFixedPoint) MulUpMagU(a, b *int256.Int) *int256.Int {
	c := a.Mul(a, b)

	if c.IsPositive() {
		return c.Add(
			c.Quo(
				c.Sub(c, s._one),
				s.ONE,
			),
			s._one,
		)
	} else if c.IsNegative() {
		return c.Sub(
			c.Quo(
				c.Add(c, s._one),
				s.ONE,
			),
			s._one,
		)
	}

	return c.Clear()
}

func (s *signedFixedPoint) DivDownMag(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	} else if a.IsZero() {
		return a, nil
	}

	aInflated, overflow := a.MulOverflow(a, s.ONE)
	if overflow {
		return nil, ErrDivInternal
	}

	return aInflated.Quo(aInflated, b), nil
}

func (s *signedFixedPoint) DivDownMagU(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}
	return a.Quo(a.Mul(a, s.ONE), b), nil
}

func (s *signedFixedPoint) DivUpMag(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	} else if a.IsZero() {
		return a, nil
	} else if b.IsNegative() {
		b.Neg(b)
		a.Neg(a)
	}

	aInflated, overflow := a.MulOverflow(a, s.ONE)
	if overflow {
		return nil, ErrDivInternal
	} else if aInflated.IsPositive() {
		return aInflated.Add(
			aInflated.Quo(
				aInflated.Sub(aInflated, s._one),
				b,
			),
			s._one,
		), nil
	}
	return aInflated.Sub(
		aInflated.Quo(
			aInflated.Add(aInflated, s._one),
			b,
		),
		s._one,
	), nil
}

func (s *signedFixedPoint) DivUpMagU(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	} else if a.IsZero() {
		return a, nil
	} else if b.IsNegative() {
		b.Neg(b)
		a.Neg(a)
	}

	if a.IsPositive() {
		return a.Add(
			a.Quo(
				a.Sub(
					a.Mul(a, s.ONE),
					s._one,
				), b,
			),
			s._one,
		), nil
	}
	return a.Sub(
		a.Quo(
			a.Add(
				a.Mul(a, s.ONE),
				s._one,
			), b,
		),
		s._one,
	), nil
}

func (s *signedFixedPoint) MulXp(a, b *int256.Int) (*int256.Int, error) {
	c, overflow := a.MulOverflow(a, b)
	if overflow {
		return nil, ErrMulOverflow
	}
	return c.Quo(c, s.ONE_XP), nil
}

func (s *signedFixedPoint) MulXpU(a, b *int256.Int) *int256.Int {
	return a.Quo(a.Mul(a, b), s.ONE_XP)
}

func (s *signedFixedPoint) DivXp(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	} else if a.IsZero() {
		return a, nil
	}

	aInflated, overflow := a.MulOverflow(a, s.ONE_XP)
	if overflow {
		return nil, ErrDivInternal
	}
	return aInflated.Quo(aInflated, b), nil
}

func (s *signedFixedPoint) DivXpU(a, b *int256.Int) (*int256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}
	return a.Quo(a.Mul(a, s.ONE_XP), b), nil
}

func (s *signedFixedPoint) MulDownXpToNp(a, b *int256.Int) (*int256.Int, error) {
	var b1, b2 int256.Int
	b1.Quo(b, s._number_1e19)
	prod1, overflow := b1.MulOverflow(a, &b1)
	if overflow {
		return nil, ErrMulOverflow
	}
	b2.Rem(b, s._number_1e19)
	prod2, overflow := b2.MulOverflow(a, &b2)
	if overflow {
		return nil, ErrMulOverflow
	} else if !prod1.IsNegative() && !prod2.IsNegative() {
		return prod1.Quo(
			prod1.Add(
				prod1,
				prod2.Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		), nil
	}
	return prod1.Sub(
		prod1.Quo(
			prod1.Add(
				prod1.Add(
					prod1,
					prod2.Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._number_1e19,
	), nil
}

func (s *signedFixedPoint) MulDownXpToNpU(a, b *int256.Int) *int256.Int {
	var b1, b2 int256.Int
	b1.Quo(b, s._number_1e19)
	prod1 := b1.Mul(a, &b1)
	b2.Rem(b, s._number_1e19)
	prod2 := b2.Mul(a, &b2)

	if !prod1.IsNegative() && !prod2.IsNegative() {
		return prod1.Quo(
			prod1.Add(
				prod1,
				prod2.Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		)
	}
	return prod1.Sub(
		prod1.Quo(
			prod1.Add(
				prod1.Add(
					prod1,
					prod2.Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._one,
	)
}

func (s *signedFixedPoint) MulUpXpToNp(a, b *int256.Int) (*int256.Int, error) {
	var b1, b2 int256.Int
	b1.Quo(b, s._number_1e19)
	prod1, overflow := b1.MulOverflow(a, &b1)
	if overflow {
		return nil, ErrMulOverflow
	}
	b2.Rem(b, s._number_1e19)
	prod2, overflow := b2.MulOverflow(a, &b2)
	if overflow {
		return nil, ErrMulOverflow
	} else if !prod1.IsPositive() && !prod2.IsPositive() {
		return prod1.Quo(
			prod1.Add(
				prod1,
				prod2.Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		), nil
	}

	return prod1.Add(
		prod1.Quo(
			prod1.Sub(
				prod1.Add(
					prod1,
					prod2.Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._one,
	), nil
}

func (s *signedFixedPoint) MulUpXpToNpU(a, b *int256.Int) *int256.Int {
	var b1, b2 int256.Int
	b1.Quo(b, s._number_1e19)
	prod1 := b1.Mul(a, &b1)
	b2.Rem(b, s._number_1e19)
	prod2 := b2.Mul(a, &b2)

	if !prod1.IsPositive() && !prod2.IsPositive() {
		return prod1.Quo(
			prod1.Add(
				prod1,
				prod2.Quo(prod2, s._number_1e19),
			),
			s._number_1e19,
		)
	}
	return prod1.Add(
		prod1.Quo(
			prod1.Sub(
				prod1.Add(
					prod1,
					prod2.Quo(prod2, s._number_1e19),
				),
				s._one,
			),
			s._number_1e19,
		),
		s._one,
	)
}

func (s *signedFixedPoint) Complement(x *int256.Int) *int256.Int {
	if x.Gte(s.ONE) || !x.IsPositive() {
		return x.Clear()
	}
	return x.Sub(s.ONE, x)
}
