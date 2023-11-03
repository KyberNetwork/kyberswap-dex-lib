package uniswapv2

import "math/big"

type ReserveData struct {
	Reserve0    *big.Int
	Reserve1    *big.Int
	BlockNumber uint64
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil && d.BlockNumber == 0
}

type StaticExtra struct {
	Fee          int64 `json:"fee"`
	FeePrecision int64 `json:"feePrecision"`
}

type PoolMeta struct {
	Fee          int64  `json:"fee"`
	FeePrecision int64  `json:"feePrecision"`
	BlockNumber  uint64 `json:"blockNumber"`
}
