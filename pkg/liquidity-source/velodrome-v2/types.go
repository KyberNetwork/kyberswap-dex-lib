package velodromev2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type PoolStaticExtra struct {
	FeePrecision uint64       `json:"feePrecision"`
	Decimal0     *uint256.Int `json:"decimal0"`
	Decimal1     *uint256.Int `json:"decimal1"`
	Stable       bool         `json:"stable"`
	DecBig       *big.Int     `json:"decBig"`
}

type PoolExtra struct {
	IsPaused bool   `json:"isPaused"`
	Fee      uint64 `json:"fee"`
}

type PoolMeta struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
	BlockNumber  uint64 `json:"blockNumber"`
}

type PoolMetadata struct {
	Dec0 *big.Int
	Dec1 *big.Int
	R0   *big.Int
	R1   *big.Int
	St   bool
	T0   common.Address
	T1   common.Address
}

type ReserveData struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type GetReservesResult struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast *big.Int
}

type PoolFactoryData struct {
	AllPairsLength *big.Int
	IsPaused       bool
}
