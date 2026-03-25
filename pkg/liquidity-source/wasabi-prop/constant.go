package wasabiprop

import "errors"

const (
	DexType    = "wasabi-prop"
	defaultGas = 200_000
	sampleSize = 15
)

// reserveSampleBps defines sample points as basis-point fractions of the input token's reserve.
// Fine-grained at the low end (0.1%–5%) to capture small-trade pricing, then coarser steps up to 100%.
var reserveSampleBps = []int{
	10, 50, 100, 250, 500, 1000, 1500, 2000, 2500, 3000,
	4000, 5000, 6000, 7000, 8000, 9000, 10000,
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
