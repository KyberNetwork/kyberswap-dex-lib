package dmm

import "math/big"

type TradeInfo struct {
	Reserve0       *big.Int
	Reserve1       *big.Int
	VReserve0      *big.Int
	VReserve1      *big.Int
	FeeInPrecision *big.Int
}

type ExtraField struct {
	VReserves      []string `json:"vReserves"`
	FeeInPrecision string   `json:"feeInPrecision"`
}
