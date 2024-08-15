package vaultT1

import "github.com/ethereum/go-ethereum/common"

type SwapPath struct {
	Protocol common.Address `json:"protocol"`
	TokenIn  common.Address `json:"tokenIn"`
	TokenOut common.Address `json:"tokenOut"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type Gas struct {
	Liquidate int64
}
