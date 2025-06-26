package v3

import (
	"math/big"
)

type Extra struct {
	IsActive bool `json:"isActive"`
	IsFrozen bool `json:"isFrozen"`
	IsPaused bool `json:"isPaused"`
}

type StaticExtra struct {
	PoolAddress string `json:"poolAddress"`
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
	IsSupply    bool   `json:"isSupply"`
	PoolAddress string `json:"poolAddress"`
}

type PoolMeta struct {
	BlockNumber uint64
}
