package baseline

import "math/big"

func nonNilBI(x *big.Int) *big.Int {
	if x == nil {
		return new(big.Int)
	}
	return x
}
