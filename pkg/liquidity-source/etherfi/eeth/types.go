package eeth

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	TotalPooledEther *big.Int `json:"totalPooledEther"`
	TotalShares      *big.Int `json:"totalShares"`
}
