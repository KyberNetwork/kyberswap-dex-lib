package brownfiv2

import (
	"math/big"

	"github.com/holiman/uint256"
)

type GetReservesResult struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

type Extra struct {
	Fee     uint64          `json:"f,omitempty"`
	Lambda  uint64          `json:"l,omitempty"`
	Kappa   *uint256.Int    `json:"k,omitempty"`
	OPrices [2]*uint256.Int `json:"p,omitempty"`
}

type PoolMeta struct {
	Fee uint64 `json:"fee"`
}
