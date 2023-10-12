package zkswapfinance

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type Meta struct {
	SwapFee      uint64 `json:"swapFee"`
	FeePrecision uint64 `json:"feePrecision"`
}

type ReservesAndParameters struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
	SwapFee  uint16
}

type StaticExtra struct {
	FeePrecision uint64 `json:"feePrecision"`
}
