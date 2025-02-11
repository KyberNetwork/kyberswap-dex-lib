package unieth

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused         bool     `json:"paused"`
	TotalSupply    *big.Int `json:"totalSupply"`
	CurrentReserve *big.Int `json:"currentReserve"`
}

type Gas struct {
	Mint int64
}
