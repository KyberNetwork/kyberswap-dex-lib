package fraxswap

import (
	"math/big"
)

type Extra struct {
	Reserve0 *big.Int `json:"reserve0"`
	Reserve1 *big.Int `json:"reserve1"`
	Fee      *big.Int `json:"fee"`
}

type GetReserveAfterTwammOutput struct {
	Reserve0                  *big.Int
	Reserve1                  *big.Int
	LastVirtualOrderTimestamp *big.Int
	TwammReserve0             *big.Int
	TwammReserve1             *big.Int
}

type FeeOutput struct {
	Fee *big.Int
}
