package zkswapfinance

import "math/big"

type Reserves struct {
	Reserve0 *big.Int `abi:""`
	Reserve1 *big.Int `abi:""`
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
