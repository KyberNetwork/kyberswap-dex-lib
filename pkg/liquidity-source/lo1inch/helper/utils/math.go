package utils

import "math/big"

type Rounding int

const (
	Ceil Rounding = iota
	Floor
)

// MulDiv performs multiplication and division of big integers with specified rounding.
func MulDiv(a, b, x *big.Int, rounding Rounding) *big.Int {
	// res = (a * b) / x
	res := new(big.Int).Mul(a, b)
	res.Div(res, x)

	// Check if the rounding is Ceil and there is a remainder
	if rounding == Ceil {
		remainder := new(big.Int).Mul(a, b)
		remainder.Mod(remainder, x)
		if remainder.Cmp(big.NewInt(0)) > 0 {
			res.Add(res, big.NewInt(1))
		}
	}

	return res
}
