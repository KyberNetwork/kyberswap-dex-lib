package sd59x18

import "math/big"

func (z *SD59x18) Add(x, y *SD59x18) *SD59x18 {
	value := new(big.Int).Add(x.value, y.value)
	z.value = value
	return z
}

func (z *SD59x18) Sub(x, y *SD59x18) *SD59x18 {
	value := new(big.Int).Sub(x.value, y.value)
	z.value = value
	return z
}

func (z *SD59x18) Lt(x, y *SD59x18) bool {
	return x.value.Cmp(y.value) < 0
}

func (z *SD59x18) Gt(x, y *SD59x18) bool {
	return x.value.Cmp(y.value) > 0
}

func (z *SD59x18) Ternary(cond bool, x, y *SD59x18) *SD59x18 {
	if cond {
		z.value.Set(x.value)
	} else {
		z.value.Set(y.value)
	}

	return z
}
