package pufeth

import "github.com/holiman/uint256"

type PoolExtra struct {
	TotalSupply      *uint256.Int `json:"totalSupply"`
	TotalAssets      *uint256.Int `json:"totalAssets"`
	TotalPooledEther *uint256.Int `json:"totalPooledEther"`
	TotalShares      *uint256.Int `json:"totalShares"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
