package weeth

import "math/big"

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type PoolExtra struct {
	TotalPooledEther *big.Int `json:"totalPooledEther"`
	TotalShares      *big.Int `json:"totalShares"`
}
