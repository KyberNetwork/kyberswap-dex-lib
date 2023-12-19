package math

import (
	"errors"

	"github.com/holiman/uint256"
)

type fixedPointCalculator struct {
	result *uint256.Int
	err    error
}

type Operator string

var (
	OperatorAdd     Operator = "add"
	OperatorSub     Operator = "sub"
	OperatorMulUp   Operator = "mulUp"
	OperatorMulDown Operator = "mulDown"
	OperatorDivUp   Operator = "divUp"
	OperatorDivDown Operator = "divDown"
)

var (
	ErrUnsupportedOperator = errors.New("unsupported operator")
)

func NewCalculator(value *uint256.Int) *fixedPointCalculator {
	return &fixedPointCalculator{
		result: value,
	}
}
func (c *fixedPointCalculator) AddWith(right *fixedPointCalculator) *fixedPointCalculator {
	return c.executeWith(OperatorAdd, right)
}

func (c *fixedPointCalculator) SubWith(right *fixedPointCalculator) *fixedPointCalculator {
	return c.executeWith(OperatorSub, right)
}

func (c *fixedPointCalculator) MulUpWith(right *fixedPointCalculator) *fixedPointCalculator {
	return c.executeWith(OperatorMulUp, right)
}

func (c *fixedPointCalculator) DivDownWith(right *fixedPointCalculator) *fixedPointCalculator {
	return c.executeWith(OperatorDivDown, right)
}

func (c *fixedPointCalculator) DivUpWith(right *fixedPointCalculator) *fixedPointCalculator {
	return c.executeWith(OperatorDivUp, right)
}

func (c *fixedPointCalculator) MulDownWith(right *fixedPointCalculator) *fixedPointCalculator {
	return c.executeWith(OperatorMulDown, right)
}

func (c *fixedPointCalculator) Add(target *uint256.Int) *fixedPointCalculator {
	return c.execute(OperatorAdd, target)
}

func (c *fixedPointCalculator) Sub(target *uint256.Int) *fixedPointCalculator {
	return c.execute(OperatorSub, target)
}

func (c *fixedPointCalculator) MulUp(target *uint256.Int) *fixedPointCalculator {
	return c.execute(OperatorMulUp, target)
}

func (c *fixedPointCalculator) DivDown(target *uint256.Int) *fixedPointCalculator {
	return c.execute(OperatorDivDown, target)
}

func (c *fixedPointCalculator) DivUp(target *uint256.Int) *fixedPointCalculator {
	return c.execute(OperatorDivUp, target)
}

func (c *fixedPointCalculator) MulDown(target *uint256.Int) *fixedPointCalculator {
	return c.execute(OperatorMulDown, target)
}

func (c *fixedPointCalculator) execute(operator Operator, target *uint256.Int) *fixedPointCalculator {
	if c.err != nil {
		return c
	}

	switch operator {
	case OperatorAdd:
		c.result, c.err = GyroFixedPoint.Add(c.result, target)

	case OperatorSub:
		c.result, c.err = GyroFixedPoint.Sub(c.result, target)

	case OperatorMulUp:
		c.result, c.err = GyroFixedPoint.MulUp(c.result, target)

	case OperatorMulDown:
		c.result, c.err = GyroFixedPoint.MulDown(c.result, target)

	case OperatorDivUp:
		c.result, c.err = GyroFixedPoint.DivUp(c.result, target)

	case OperatorDivDown:
		c.result, c.err = GyroFixedPoint.DivDown(c.result, target)

	default:
		c.err = ErrUnsupportedOperator
	}

	return c
}

func (c *fixedPointCalculator) executeWith(operator Operator, right *fixedPointCalculator) *fixedPointCalculator {
	if c.err != nil {
		return c
	}

	rightResult, err := right.Result()
	if err != nil {
		c.err = err
		return c
	}

	return c.execute(operator, rightResult)
}

func (c *fixedPointCalculator) Result() (*uint256.Int, error) {
	return c.result, c.err
}
