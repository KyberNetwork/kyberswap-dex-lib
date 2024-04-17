package sweth

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused         bool     `json:"paused"`
	SWETHToETHRate *big.Int `json:"swETHToETHRate"`
}
