package math

import (
	"github.com/holiman/uint256"
)

func U256ToFloatBaseX128(x128 *uint256.Int) float64 {
	return x128.Float64() / 340282366920938463463374607431768211456
}
