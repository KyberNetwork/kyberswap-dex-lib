package wasabiprop

import "errors"

const (
	DexType    = "wasabi-prop"
	defaultGas = 200_000
	sampleSize = 15
)

// reserveSampleBps defines sample points as basis-point fractions of the input token's reserve.
// 5% to 100% in 5% steps gives fine-grained coverage near the liquidity boundary.
var reserveSampleBps = []int{
	500, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000,
	5500, 6000, 6500, 7000, 7500, 8000, 8500, 9000, 9500, 10000,
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
