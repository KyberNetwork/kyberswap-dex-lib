package sd59x18

import (
	"math/big"
)

func ConvertToSD59x18(x *big.Int) (SD59x18, error) {
	if x.Cmp(new(big.Int).Div(uMinSD59x18, uUnit)) < 0 {
		return nil, ErrMathSD59x18ConvertUnderflow
	}
	if x.Cmp(new(big.Int).Div(uMaxSD59x18, uUnit)) > 0 {
		return nil, ErrMathSD59x18ConvertOverflow
	}
	result := new(big.Int).Mul(x, uUnit)
	return result, nil
}

func ConvertToBI(x SD59x18) *big.Int {
	return new(big.Int).Div(x, uUnit)
}

func Sd(x *big.Int) SD59x18 {
	return x
}

func Wrap(x *big.Int) SD59x18 {
	return x
}

func Unwrap(x SD59x18) *big.Int {
	return x
}
