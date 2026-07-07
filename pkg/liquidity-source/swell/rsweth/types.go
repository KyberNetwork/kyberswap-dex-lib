package rsweth

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused          bool     `json:"paused"`
	ETHToRswETHRate *big.Int `json:"ethToRswETHRate"`
}
