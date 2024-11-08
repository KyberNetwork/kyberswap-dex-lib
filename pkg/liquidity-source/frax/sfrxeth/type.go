package sfrxeth

import (
	"github.com/holiman/uint256"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	SubmitPaused bool         `json:"submitPaused"`
	TotalSupply  *uint256.Int `json:"totalSupply"`
	TotalAssets  *uint256.Int `json:"totalAssets"`
}

type Gas struct {
	SubmitAndDeposit int64
}

type PoolItem struct {
	FrxETHMinterAddress string `json:"frxETHMinter"`
	FrxETHAddress       string `json:"frxETH"`
	SfrxETHAddress      string `json:"sfrxETH"`
}
