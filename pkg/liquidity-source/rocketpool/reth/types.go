package reth

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	DepositEnabled         bool     `json:"depositEnabled"`
	MinimumDeposit         *big.Int `json:"minimumDeposit"`
	MaximumDepositPoolSize *big.Int `json:"maximumDepositPoolSize"`
	AssignDepositsEnabled  bool     `json:"assignDepositsEnabled"`
	DepositFee             *big.Int `json:"depositFee"`
	Balance                *big.Int `json:"balance"`
	EffectiveCapacity      *big.Int `json:"effectiveCapacity"`
	TotalETHBalance        *big.Int `json:"totalETHBalance"`
	TotalRETHSupply        *big.Int `json:"totalRETHSupply"`
	ExcessBalance          *big.Int `json:"excessBalance"`
	RETHBalance            *big.Int `json:"rETHBalance"`
}
