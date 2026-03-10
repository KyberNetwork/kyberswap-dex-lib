package poe

import (
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Oracle string `json:"oracle"`
}

type Extra struct {
	Price   *uint256.Int `json:"p"`
	FeeHbps *uint256.Int `json:"fee_hbps"`
	Alpha   *uint256.Int `json:"alpha"`
	Expiry  uint64       `json:"exp"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"bN"`
	IsXtoY      bool   `json:"isXtoY,omitempty"`
}

type virtualReserves struct {
	xv *uint256.Int
	yv *uint256.Int
}
