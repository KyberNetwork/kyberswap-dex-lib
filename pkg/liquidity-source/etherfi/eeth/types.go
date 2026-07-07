package eeth

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
	Pool        string `json:"pool"`
	EETHToken   string `json:"eethToken"`
}

type PoolExtra struct {
	TotalPooledEther *big.Int `json:"totalPooledEther"`
	TotalShares      *big.Int `json:"totalShares"`
}
