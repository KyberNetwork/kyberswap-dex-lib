package maverickv2

import (
	"math/big"

	"github.com/holiman/uint256"
)

type State struct {
	ReserveA           *big.Int `json:"reserveA"`
	ReserveB           *big.Int `json:"reserveB"`
	LastTimestamp      int64    `json:"lastTimestamp"`
	LastTwaD8          int64    `json:"lastTwaD8"`
	LastLogPriceD8     int64    `json:"lastLogPriceD8"`
	ActiveTick         int32    `json:"activeTick"`
	IsLocked           bool     `json:"isLocked"`
	BinCounter         uint32   `json:"binCounter"`
	ProtocolFeeRatioD3 uint8    `json:"protocolFeeRatioD3"`
	FeeAIn             uint64   `json:"feeAIn"` // Fee for tokenA -> tokenB swaps
	FeeBIn             uint64   `json:"feeBIn"` // Fee for tokenB -> tokenA swaps
}

// TwaData represents Time-Weighted Average data for the pool
type TwaData struct {
	LastTimestamp int64        `json:"lastTimestamp"`
	LastTwaD8     int64        `json:"lastTwaD8"`
	LookbackSec   int64        `json:"lookbackSec"`
	AccumValue    *uint256.Int `json:"accumValue"`
}

// MoveBinsParams contains parameters needed for the moveBins operation
type MoveBinsParams struct {
	StartingTick int32
	EndTick      int32
	OldTwaD8     int64
	NewTwaD8     int64
	Threshold    *uint256.Int
}

// MoveData contains parameters for bin movement operations
type MoveData struct {
	Kind            uint8           `json:"kind"`
	TickSearchStart int32           `json:"tickSearchStart"`
	TickSearchEnd   int32           `json:"tickSearchEnd"`
	TickLimit       int32           `json:"tickLimit"`
	FirstBinTick    int32           `json:"firstBinTick"`
	FirstBinId      uint32          `json:"firstBinId"`
	MergeBinBalance *uint256.Int    `json:"mergeBinBalance"`
	TotalReserveA   *uint256.Int    `json:"totalReserveA"`
	TotalReserveB   *uint256.Int    `json:"totalReserveB"`
	MergeBins       map[uint32]bool `json:"mergeBins"`
	Counter         uint32          `json:"counter"`
}

type StaticExtra struct {
	TickSpacing uint32 `json:"tickSpacing"`
}
