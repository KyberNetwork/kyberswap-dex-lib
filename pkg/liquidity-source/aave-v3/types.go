package aavev3

import (
	"math/big"
)

type Extra struct {
	IsActive bool `json:"isActive"`
	IsFrozen bool `json:"isFrozen"`
	IsPaused bool `json:"isPaused"`
}

type StaticExtra struct {
	AavePoolAddress string `json:"aavePoolAddress"`
}

type RPCData struct {
	Configuration ReserveConfigurationMap
	BlockNumber   uint64
}

type ReserveConfigurationMap struct {
	Data struct {
		Data *big.Int
	}
}

type SwapInfo struct {
	IsSupply        bool   `json:"isSupply"`
	AavePoolAddress string `json:"aaveV3PoolAddress"`
}

type PoolMeta struct {
	BlockNumber uint64
}
