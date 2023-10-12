package zkswapfinance

import "math/big"

type Reserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast *big.Int
}

type Metadata struct {
	Offset int `json:"offset"`
}

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}

type Meta struct {
	SwapFee string `json:"swapFee"`
}
