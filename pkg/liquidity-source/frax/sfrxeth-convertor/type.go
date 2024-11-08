package sfrxeth_convertor

import (
	"github.com/holiman/uint256"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	TotalSupply *uint256.Int `json:"totalSupply"`
	TotalAssets *uint256.Int `json:"totalAssets"`
}

type Gas struct {
	Deposit int64
	Redeem  int64
}

type PoolItem struct {
	FrxETHAddress  string `json:"frxETH"`
	SfrxETHAddress string `json:"sfrxETH"`
}
