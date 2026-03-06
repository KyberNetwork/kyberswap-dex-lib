package reth

import "math/big"

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber,omitempty"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type PoolExtra struct {
	DepositEnabled         bool     `json:"d,omitempty"`
	MinimumDeposit         *big.Int `json:"m,omitempty"`
	MaximumDepositPoolSize *big.Int `json:"p,omitempty"`
	AssignDepositsEnabled  bool     `json:"a,omitempty"`
	DepositFee             *big.Int `json:"f,omitempty"`
	Balance                *big.Int `json:"b,omitempty"`
	EffectiveCapacity      *big.Int `json:"c,omitempty"`
	TotalETHBalance        *big.Int `json:"e,omitempty"`
	TotalRETHSupply        *big.Int `json:"s,omitempty"`
	ExcessBalance          *big.Int `json:"x,omitempty"`
	RETHBalance            *big.Int `json:"r,omitempty"`
}
