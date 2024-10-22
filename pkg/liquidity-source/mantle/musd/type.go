package musd

import (
	"math/big"

	"github.com/holiman/uint256"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused         bool         `json:"paused"`
	OraclePrice    *uint256.Int `json:"oraclePrice"`
	PriceTimestamp uint64       `json:"priceTimeStamp"`
}

type OraclePriceData struct {
	Price     *big.Int `json:"price"`
	Timestamp *big.Int `json:"timestamp"`
}
