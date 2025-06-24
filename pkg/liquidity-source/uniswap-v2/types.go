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
	Extra
	PoolMetaGeneric
}

type PoolMetaGeneric struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
	NoFOT           bool   `json:"noFOT,omitempty"`
}
