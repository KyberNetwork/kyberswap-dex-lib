package maverickv2

import (
	"math/big"
	"github.com/holiman/uint256"
)

type State struct {
	ReserveA      *big.Int `json:"reserveA"`
	ReserveB      *big.Int `json:"reserveB"`
	LastTimestamp int64    `json:"lastTimestamp"`
}

// TwaData represents Time-Weighted Average data for the pool
type TwaData struct {
	LastTimestamp int64         `json:"lastTimestamp"`
	LastTwaD8     int64         `json:"lastTwaD8"`
	LookbackSec   int64         `json:"lookbackSec"`
	AccumValue    *uint256.Int  `json:"accumValue"`
}

// MoveBinsParams contains parameters needed for the moveBins operation
type MoveBinsParams struct {
	StartingTick int32
	EndTick      int32
	OldTwaD8     int64
	NewTwaD8     int64
	Threshold    *uint256.Int
}
