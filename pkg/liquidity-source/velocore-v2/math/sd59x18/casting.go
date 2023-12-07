package sd59x18

import "math/big"

func SD(x *big.Int) *SD59x18 {
	return &SD59x18{value: x}
}

func IntoInt256(x *SD59x18) *big.Int {
	return x.value
}
