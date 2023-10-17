package sd59x18

import "math/big"

func Add(x SD59x18, y SD59x18) SD59x18 {
	return new(big.Int).Add(x, y)
}

func Sub(x SD59x18, y SD59x18) SD59x18 {
	return new(big.Int).Sub(x, y)
}

func Lt(x SD59x18, y SD59x18) bool {
	xBI := new(big.Int).Set(x)
	yBI := new(big.Int).Set(y)
	return xBI.Cmp(yBI) < 0
}

func Zero() SD59x18 {
	return big.NewInt(0)
}
