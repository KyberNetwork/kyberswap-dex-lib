package primeeth

import "math/big"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused              bool     `json:"paused"`
	TotalAssetDeposit   *big.Int `json:"totalAssetDeposit"`
	DepositLimitByAsset *big.Int `json:"depositLimitByAsset"`
	MinAmountToDeposit  *big.Int `json:"minAmountToDeposit"`
	PrimeETHPrice       *big.Int `json:"primeETHPrice"`
}

type Gas struct {
	Deposit int64
}
