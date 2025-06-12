package math

import "math/big"

func U256ToFloatBaseX128(x128 *big.Int) float64 {
	float, _ := x128.Float64()
	return float / 340282366920938463463374607431768211456
}
