package fraxswap

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type Extra struct {
	Reserve0 *big.Int `json:"reserve0"`
	Reserve1 *big.Int `json:"reserve1"`
	Fee      *big.Int `json:"fee"`
}

type ReserveAfterTwammOutput struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

type FeeOutput struct {
	Fee *big.Int
}

type Gas struct {
	Swap int64
}

type Meta struct {
	SwapFee      uint32 `json:"swapFee"`
	FeePrecision uint32 `json:"feePrecision"`
}
