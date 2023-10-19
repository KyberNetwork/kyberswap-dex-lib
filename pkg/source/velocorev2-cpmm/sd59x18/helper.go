package sd59x18

import "math/big"

func Add(x SD59x18, y SD59x18) SD59x18 {
	return new(big.Int).Add(x, y)
}

func Sub(x SD59x18, y SD59x18) SD59x18 {
	return new(big.Int).Sub(x, y)
}

func Lt(x SD59x18, y SD59x18) bool {
	var (
		xBI *big.Int = x
		yBI *big.Int = y
	)
	return xBI.Cmp(yBI) < 0
}

func Zero() SD59x18 {
	return bigint0
}
