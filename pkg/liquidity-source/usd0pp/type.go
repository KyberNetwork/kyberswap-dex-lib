package usd0pp

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused      bool     `json:"paused"`
	EndTime     int64    `json:"endTime"`
	StartTime   int64    `json:"startTime"`
	TotalSupply *big.Int `json:"totalSupply"`
}
