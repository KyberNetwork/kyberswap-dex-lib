package syncswap

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type Extra struct {
	SwapFee0To1 *big.Int `json:"swapFee0To1"`
	SwapFee1To0 *big.Int `json:"swapFee1To0"`
}
