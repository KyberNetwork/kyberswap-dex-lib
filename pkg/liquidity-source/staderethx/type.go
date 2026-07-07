package staderethx

import (
	"github.com/holiman/uint256"
	"math/big"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused               bool         `json:"paused"`
	MinDeposit           *uint256.Int `json:"minDeposit"`
	MaxDeposit           *uint256.Int `json:"maxDeposit"`
	ReportingBlockNumber uint64       `json:"reportingBlockNumber"`
	TotalETHBalance      *uint256.Int `json:"totalETHBalance"`
	TotalETHXSupply      *uint256.Int `json:"totalETHXSupply"`
}

type Gas struct {
	Deposit int64
}

type StaderOracleExchangeRate struct {
	ReportingBlockNumber *big.Int `json:"reportingBlockNumber"`
	TotalETHBalance      *big.Int `json:"totalETHBalance"`
	TotalETHXSupply      *big.Int `json:"totalETHXSupply"`
}
