package sd59x18

import "math/big"

func (z *SD59x18) Add(x, y *SD59x18) *SD59x18 {
	z.value = new(big.Int).Add(x.value, y.value)
	return z
}

func (z *SD59x18) Sub(x, y *SD59x18) *SD59x18 {
	z.value = new(big.Int).Sub(x.value, y.value)
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
		z.value = new(big.Int).Set(x.value)
	} else {
		z.value = new(big.Int).Set(y.value)
	}
	return z
}
