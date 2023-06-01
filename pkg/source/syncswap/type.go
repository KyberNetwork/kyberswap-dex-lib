package syncswap

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type ExtraClassicPool struct {
	SwapFee0To1 *big.Int `json:"swapFee0To1"`
	SwapFee1To0 *big.Int `json:"swapFee1To0"`
}

type ExtraStablePool struct {
	SwapFee0To1 *big.Int `json:"swapFee0To1"`
	SwapFee1To0 *big.Int `json:"swapFee1To0"`

	Token0PrecisionMultiplier *big.Int `json:"token0PrecisionMultiplier"`
	Token1PrecisionMultiplier *big.Int `json:"token1PrecisionMultiplier"`
}
