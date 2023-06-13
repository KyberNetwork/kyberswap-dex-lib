package syncswapclassic

import "math/big"

type Gas struct {
	Swap int64
}

type ExtraClassicPool struct {
	SwapFee0To1 *big.Int `json:"swapFee0To1"`
	SwapFee1To0 *big.Int `json:"swapFee1To0"`
}
