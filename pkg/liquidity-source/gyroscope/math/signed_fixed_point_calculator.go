package math

import (
	"math/big"
)

type SignedFixedPointCalculator struct {
	result *big.Int
	err    error
}

type SignedFixedPointOperator string

const (
	SignedFixedPointOperatorAdd            SignedFixedPointOperator = "add"
	SignedFixedPointOperatorAddMag         SignedFixedPointOperator = "addMag"
	SignedFixedPointOperatorSub            SignedFixedPointOperator = "sub"
	SignedFixedPointOperatorMulDownMag     SignedFixedPointOperator = "mulDownMag"
	SignedFixedPointOperatorMulDownMagU    SignedFixedPointOperator = "mulDownMagU"
	SignedFixedPointOperatorMulUpMag       SignedFixedPointOperator = "mulUpMag"
	SignedFixedPointOperatorMulUpMagU      SignedFixedPointOperator = "mulUpMagU"
	SignedFixedPointOperatorDivDownMag     SignedFixedPointOperator = "divDownMag"
	SignedFixedPointOperatorDivDownMagU    SignedFixedPointOperator = "divDownMagU"
	SignedFixedPointOperatorDivUpMag       SignedFixedPointOperator = "divUpMag"
	SignedFixedPointOperatorDivUpMagU      SignedFixedPointOperator = "divUpMagU"
	SignedFixedPointOperatorMulXp          SignedFixedPointOperator = "mulXp"
	SignedFixedPointOperatorMulXpU         SignedFixedPointOperator = "mulXpU"
	SignedFixedPointOperatorDivXp          SignedFixedPointOperator = "divXp"
	SignedFixedPointOperatorDivXpU         SignedFixedPointOperator = "divXpU"
	SignedFixedPointOperatorMulDownXpToNp  SignedFixedPointOperator = "mulDownXpToNp"
	SignedFixedPointOperatorMulDownXpToNpU SignedFixedPointOperator = "mulDownXpToNpU"
	SignedFixedPointOperatorMulUpXpToNp    SignedFixedPointOperator = "mulUpXpToNp"
	SignedFixedPointOperatorMulUpXpToNpU   SignedFixedPointOperator = "mulUpXpToNpU"
	SignedFixedPointOperatorComplement     SignedFixedPointOperator = "complement"
)

func NewSignedFixedPointCalculator(value *big.Int) *SignedFixedPointCalculator {
	return &SignedFixedPointCalculator{
		result: value,
	}
}

func (c *SignedFixedPointCalculator) Result() (*big.Int, error) {
	return c.result, c.err
}

func (c *SignedFixedPointCalculator) Ternary(condition bool, trueValue, falseValue *big.Int) *SignedFixedPointCalculator {
	if condition {
		c.result = trueValue
	} else {
		c.result = falseValue
	}
	return c
}

func (c *SignedFixedPointCalculator) Add(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorAdd, other)
}

func (c *SignedFixedPointCalculator) AddMag(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorAddMag, other)
}

func (c *SignedFixedPointCalculator) Sub(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorSub, other)
}

func (c *SignedFixedPointCalculator) MulDownMag(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulDownMag, other)
}

func (c *SignedFixedPointCalculator) MulDownMagU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulDownMagU, other)
}

func (c *SignedFixedPointCalculator) MulUpMag(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulUpMag, other)
}

func (c *SignedFixedPointCalculator) MulUpMagU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulUpMagU, other)
}

func (c *SignedFixedPointCalculator) DivDownMag(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorDivDownMag, other)
}

func (c *SignedFixedPointCalculator) DivDownMagU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorDivDownMagU, other)
}

func (c *SignedFixedPointCalculator) DivUpMag(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorDivUpMag, other)
}

func (c *SignedFixedPointCalculator) DivUpMagU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorDivUpMagU, other)
}

func (c *SignedFixedPointCalculator) MulXp(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulXp, other)
}

func (c *SignedFixedPointCalculator) MulXpU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulXpU, other)
}

func (c *SignedFixedPointCalculator) DivXp(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorDivXp, other)
}

func (c *SignedFixedPointCalculator) DivXpU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorDivXpU, other)
}

func (c *SignedFixedPointCalculator) MulDownXpToNp(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulDownXpToNp, other)
}

func (c *SignedFixedPointCalculator) MulDownXpToNpU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulDownXpToNpU, other)
}

func (c *SignedFixedPointCalculator) MulUpXpToNp(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulUpXpToNp, other)
}

func (c *SignedFixedPointCalculator) MulUpXpToNpU(other *big.Int) *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorMulUpXpToNpU, other)
}

func (c *SignedFixedPointCalculator) Complement() *SignedFixedPointCalculator {
	return c.execute(SignedFixedPointOperatorComplement, nil)
}

func (c *SignedFixedPointCalculator) TernaryWith(condition bool, trueValue, falseValue *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	if condition {
		c = trueValue
	} else {
		c = falseValue
	}
	return c
}

func (c *SignedFixedPointCalculator) AddWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorAdd, right)
}

func (c *SignedFixedPointCalculator) AddMagWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorAddMag, right)
}

func (c *SignedFixedPointCalculator) SubWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorSub, right)
}

func (c *SignedFixedPointCalculator) MulDownMagWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulDownMag, right)
}

func (c *SignedFixedPointCalculator) MulDownMagUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulDownMagU, right)
}

func (c *SignedFixedPointCalculator) MulUpMagWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulUpMag, right)
}

func (c *SignedFixedPointCalculator) MulUpMagUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulUpMagU, right)
}

func (c *SignedFixedPointCalculator) DivDownMagWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorDivDownMag, right)
}

func (c *SignedFixedPointCalculator) DivDownMagUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorDivDownMagU, right)
}

func (c *SignedFixedPointCalculator) DivUpMagWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorDivUpMag, right)
}

func (c *SignedFixedPointCalculator) DivUpMagUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorDivUpMagU, right)
}

func (c *SignedFixedPointCalculator) MulXpWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulXp, right)
}

func (c *SignedFixedPointCalculator) MulXpUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulXpU, right)
}

func (c *SignedFixedPointCalculator) DivXpWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorDivXp, right)
}

func (c *SignedFixedPointCalculator) DivXpUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorDivXpU, right)
}

func (c *SignedFixedPointCalculator) MulDownXpToNpWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulDownXpToNp, right)
}

func (c *SignedFixedPointCalculator) MulDownXpToNpUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulDownXpToNpU, right)
}

func (c *SignedFixedPointCalculator) MulUpXpToNpWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulUpXpToNp, right)
}

func (c *SignedFixedPointCalculator) MulUpXpToNpUWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorMulUpXpToNpU, right)
}

func (c *SignedFixedPointCalculator) ComplementWith(right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	return c.executeWith(SignedFixedPointOperatorComplement, right)
}

func (c *SignedFixedPointCalculator) executeWith(op SignedFixedPointOperator, right *SignedFixedPointCalculator) *SignedFixedPointCalculator {
	if c.err != nil {
		return c
	}

	rightResult, err := right.Result()
	if err != nil {
		c.err = err
		return c
	}

	return c.execute(op, rightResult)
}

func (c *SignedFixedPointCalculator) execute(op SignedFixedPointOperator, target *big.Int) *SignedFixedPointCalculator {
	if c.err != nil {
		return c
	}

	switch op {
	case SignedFixedPointOperatorAdd:
		c.result, c.err = SignedFixedPoint.Add(c.result, target)
	case SignedFixedPointOperatorAddMag:
		c.result, c.err = SignedFixedPoint.AddMag(c.result, target)
	case SignedFixedPointOperatorSub:
		c.result, c.err = SignedFixedPoint.Sub(c.result, target)
	case SignedFixedPointOperatorMulDownMag:
		c.result, c.err = SignedFixedPoint.MulDownMag(c.result, target)
	case SignedFixedPointOperatorMulDownMagU:
		c.result = SignedFixedPoint.MulDownMagU(c.result, target)
	case SignedFixedPointOperatorMulUpMag:
		c.result, c.err = SignedFixedPoint.MulUpMag(c.result, target)
	case SignedFixedPointOperatorMulUpMagU:
		c.result = SignedFixedPoint.MulUpMagU(c.result, target)
	case SignedFixedPointOperatorDivDownMag:
		c.result, c.err = SignedFixedPoint.DivDownMag(c.result, target)
	case SignedFixedPointOperatorDivDownMagU:
		c.result, c.err = SignedFixedPoint.DivDownMagU(c.result, target)
	case SignedFixedPointOperatorDivUpMag:
		c.result, c.err = SignedFixedPoint.DivUpMag(c.result, target)
	case SignedFixedPointOperatorDivUpMagU:
		c.result, c.err = SignedFixedPoint.DivUpMagU(c.result, target)
	case SignedFixedPointOperatorMulXp:
		c.result, c.err = SignedFixedPoint.MulXp(c.result, target)
	case SignedFixedPointOperatorMulXpU:
		c.result = SignedFixedPoint.MulXpU(c.result, target)
	case SignedFixedPointOperatorDivXp:
		c.result, c.err = SignedFixedPoint.DivXp(c.result, target)
	case SignedFixedPointOperatorDivXpU:
		c.result, c.err = SignedFixedPoint.DivXpU(c.result, target)
	case SignedFixedPointOperatorMulDownXpToNp:
		c.result, c.err = SignedFixedPoint.MulDownXpToNp(c.result, target)
	case SignedFixedPointOperatorMulDownXpToNpU:
		c.result = SignedFixedPoint.MulDownXpToNpU(c.result, target)
	case SignedFixedPointOperatorMulUpXpToNp:
		c.result, c.err = SignedFixedPoint.MulUpXpToNp(c.result, target)
	case SignedFixedPointOperatorMulUpXpToNpU:
		c.result = SignedFixedPoint.MulUpXpToNpU(c.result, target)
	case SignedFixedPointOperatorComplement:
		c.result = SignedFixedPoint.Complement(c.result)
	default:
		c.err = ErrUnsupportedOperator
	}

	return c
}
