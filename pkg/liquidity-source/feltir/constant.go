package feltir

import "errors"

const (
	DexType    = "feltir"
	defaultGas = 200_000
	sampleSize = 15
)

// reserveSampleBps defines sample points as basis-point fractions of the input token's reserve (0.1%–99%).
// Fine-grained at the low end to capture small-trade pricing, coarser toward capacity.
// 100% is intentionally excluded: quoting the full reserve always reverts on-chain.
var reserveSampleBps = []int{
	10, 50, 250, 500, 1000, 2000, 3000, 5000, 7000, 9000, 9900,
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
