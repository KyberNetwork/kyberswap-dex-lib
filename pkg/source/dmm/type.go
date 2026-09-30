package dmm

import "github.com/holiman/uint256"

type PoolModelReserves []string

type Extra struct {
	VReserves      PoolModelReserves `json:"vReserves"`
	FeeInPrecision string            `json:"feeInPrecision"`
}

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}

type Metadata struct {
	Offset int `json:"offset"`
}

type TradeInfo struct {
	Reserve0       *uint256.Int
	Reserve1       *uint256.Int
	VReserve0      *uint256.Int
	VReserve1      *uint256.Int
	FeeInPrecision *uint256.Int
}

type ExtraField struct {
	VReserves      []string `json:"vReserves"`
	FeeInPrecision string   `json:"feeInPrecision"`
}
