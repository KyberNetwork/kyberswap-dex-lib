package math

import (
	"math/big"
)

func muldiv(x, y, d *big.Int, roundUp bool) (*big.Int, error) {
	if d.Sign() == 0 {
		return nil, ErrDivZero
	}

	intermediate := new(big.Int)

	if y.Cmp(One) == 0 {
		intermediate.Set(x)
	} else {
		intermediate.Mul(x, y)
	}

	quotient, remainder := intermediate.DivMod(
		intermediate,
		d,
		new(big.Int),
	)

	if roundUp && remainder.Sign() != 0 {
		quotient.Add(quotient, One)
	}

	if quotient.Cmp(TwoPow256) != -1 {
		return nil, ErrOverflow
	}

	return quotient, nil
}
