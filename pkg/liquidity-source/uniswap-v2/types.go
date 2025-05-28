package uniswapv2

import "math/big"

type ReserveData struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type Extra struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
}

type PoolMeta struct {
	Fee             uint64 `json:"fee"`
	FeePrecision    uint64 `json:"feePrecision"`
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}
