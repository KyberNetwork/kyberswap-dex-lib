package deltaswapv1

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// Sqrt babylonian method (https://en.wikipedia.org/wiki/Methods_of_computing_square_roots#Babylonian_method)
func Sqrt(y *uint256.Int) *uint256.Int {
	var x, z uint256.Int
	if y.Gt(number.Number_3) {
		z.Set(y)
		x.Div(y, number.Number_2).Add(&x, number.Number_1)
		for x.Lt(&z) {
			z.Set(&x)
			x.Div(y, &x).Add(&x, &z).Div(&x, number.Number_2)
		}
	} else if !y.IsZero() {
		z.Set(number.Number_1)
	}
	return &z
}

func Max(a *uint256.Int, b *uint256.Int) *uint256.Int {
	if a.Gt(b) {
		return a
	}
	return b
}
