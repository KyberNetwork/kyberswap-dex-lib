package ondo_usdy

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/holiman/uint256"
)

type PoolItem struct {
	ID                      string             `json:"id"`
	Type                    string             `json:"type"`
	Tokens                  []entity.PoolToken `json:"tokens"`
	RWADynamicOracleAddress string             `json:"rwaDynamicOracleAddress"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused                  bool         `json:"paused"`
	OraclePrice             *uint256.Int `json:"oraclePrice"`
	PriceTimestamp          uint64       `json:"priceTimeStamp"`
	RWADynamicOracleAddress string       `json:"rwaDynamicOracleAddress"`
}

type OraclePriceData struct {
	Price     *big.Int `json:"price"`
	Timestamp *big.Int `json:"timestamp"`
}
