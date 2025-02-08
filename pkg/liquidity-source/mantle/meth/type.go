package meth

import "github.com/holiman/uint256"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	IsStakingPaused        bool         `json:"isStakingPaused"`
	MinimumStakeBound      *uint256.Int `json:"minimumStakeBound"`
	MaximumMETHSupply      *uint256.Int `json:"maximumMETHSupply"`
	TotalControlled        *uint256.Int `json:"totalControlled"`
	ExchangeAdjustmentRate uint16       `json:"exchangeAdjustmentRate"`
	METHTotalSupply        *uint256.Int `json:"mETHTotalSupply"`
}

type Gas struct {
	Stake int64
}
