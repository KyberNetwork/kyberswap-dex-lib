package aeonvamm

import "math/big"

type Extra struct {
	Reserve0 *big.Int `json:"reserve0"`
	Reserve1 *big.Int `json:"reserve1"`
	Fee      uint64   `json:"fee"` // in bps, e.g. 30 = 0.3%
}

type PoolMeta struct {
	Fee          uint64 `json:"fee"`
	BlockNumber  uint64 `json:"blockNumber"`
}

type PoolListUpdaterMetadata struct {
	Offset int `json:"offset"`
}
