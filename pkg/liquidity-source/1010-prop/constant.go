package prop

import (
	"errors"
)

const (
	DexType    = "1010-prop"
	defaultGas = 135_000
	sampleSize = 15 // power-of-10 levels
)

var maxInSampleBps = []int{
	1000, 1500, 2200, 3200, 4000, // 10–40%
	4500, 5000, 5600, 6200, 6800, // 40–68%
	7300, 7900, 8500, 9100, 9900, // 73–99%
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
