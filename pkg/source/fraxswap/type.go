package fraxswap

import (
	"math/big"

	"github.com/holiman/uint256"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type Extra struct {
	Reserve0 *big.Int `json:"reserve0"`
	Reserve1 *big.Int `json:"reserve1"`
	Fee      *big.Int `json:"fee"`
}

type ReserveAfterTwammOutput struct {
	Reserve0 *uint256.Int
	Reserve1 *uint256.Int
}

type FeeOutput struct {
	Fee *uint256.Int
}

type Gas struct {
	Swap int64
}

type Meta struct {
	SwapFee      uint32 `json:"swapFee"`
	FeePrecision uint32 `json:"feePrecision"`
	BlockNumber  uint64 `json:"blockNumber"`
}
