package math

import (
	"errors"

	"github.com/holiman/uint256"
)

type FixedPointCalculator struct {
	result *uint256.Int
	err    error
}

type FixedPointOperator string

var (
	FixedPointOperatorAdd      FixedPointOperator = "add"
	FixedPointOperatorSub      FixedPointOperator = "sub"
	FixedPointOperatorMulUp    FixedPointOperator = "mulUp"
	FixedPointOperatorMulDown  FixedPointOperator = "mulDown"
	FixedPointOperatorDivUp    FixedPointOperator = "divUp"
	FixedPointOperatorDivDown  FixedPointOperator = "divDown"
	FixedPointOperatorMulDownU FixedPointOperator = "mulDownU"
)

var (
	ErrUnsupportedOperator = errors.New("unsupported operator")
)

func NewFixedPointCalculator(value *uint256.Int) *FixedPointCalculator {
	return &FixedPointCalculator{
		result: value,
	}
}
func (c *FixedPointCalculator) AddWith(right *FixedPointCalculator) *FixedPointCalculator {
	return c.executeWith(FixedPointOperatorAdd, right)
}

func (c *FixedPointCalculator) SubWith(right *FixedPointCalculator) *FixedPointCalculator {
	return c.executeWith(FixedPointOperatorSub, right)
}

func (c *FixedPointCalculator) MulUpWith(right *FixedPointCalculator) *FixedPointCalculator {
	return c.executeWith(FixedPointOperatorMulUp, right)
}

func (c *FixedPointCalculator) DivDownWith(right *FixedPointCalculator) *FixedPointCalculator {
	return c.executeWith(FixedPointOperatorDivDown, right)
}

func (c *FixedPointCalculator) DivUpWith(right *FixedPointCalculator) *FixedPointCalculator {
	return c.executeWith(FixedPointOperatorDivUp, right)
}

func (c *FixedPointCalculator) MulDownWith(right *FixedPointCalculator) *FixedPointCalculator {
	return c.executeWith(FixedPointOperatorMulDown, right)
}

func (c *FixedPointCalculator) Add(target *uint256.Int) *FixedPointCalculator {
	return c.execute(FixedPointOperatorAdd, target)
}

func (c *FixedPointCalculator) Sub(target *uint256.Int) *FixedPointCalculator {
	return c.execute(FixedPointOperatorSub, target)
}

func (c *FixedPointCalculator) MulUp(target *uint256.Int) *FixedPointCalculator {
	return c.execute(FixedPointOperatorMulUp, target)
}

func (c *FixedPointCalculator) DivDown(target *uint256.Int) *FixedPointCalculator {
	return c.execute(FixedPointOperatorDivDown, target)
}

func (c *FixedPointCalculator) DivUp(target *uint256.Int) *FixedPointCalculator {
	return c.execute(FixedPointOperatorDivUp, target)
}

func (c *FixedPointCalculator) MulDown(target *uint256.Int) *FixedPointCalculator {
	return c.execute(FixedPointOperatorMulDown, target)
}

func (c *FixedPointCalculator) MulDownU(target *uint256.Int) *FixedPointCalculator {
	return c.execute(FixedPointOperatorMulDownU, target)
}

func (c *FixedPointCalculator) execute(operator FixedPointOperator, target *uint256.Int) *FixedPointCalculator {
	if c.err != nil {
		return c
	}

	switch operator {
	case FixedPointOperatorAdd:
		c.result, c.err = GyroFixedPoint.Add(c.result, target)

	case FixedPointOperatorSub:
		c.result, c.err = GyroFixedPoint.Sub(c.result, target)

	case FixedPointOperatorMulUp:
		c.result, c.err = GyroFixedPoint.MulUp(c.result, target)

	case FixedPointOperatorMulDown:
		c.result, c.err = GyroFixedPoint.MulDown(c.result, target)

	case FixedPointOperatorDivUp:
		c.result, c.err = GyroFixedPoint.DivUp(c.result, target)

	case FixedPointOperatorDivDown:
		c.result, c.err = GyroFixedPoint.DivDown(c.result, target)

	case FixedPointOperatorMulDownU:
		c.result = GyroFixedPoint.MulDownU(c.result, target)

	default:
		c.err = ErrUnsupportedOperator
	}

	return c
}

func (c *FixedPointCalculator) executeWith(operator FixedPointOperator, right *FixedPointCalculator) *FixedPointCalculator {
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

func (c *FixedPointCalculator) Result() (*uint256.Int, error) {
	return c.result, c.err
}
