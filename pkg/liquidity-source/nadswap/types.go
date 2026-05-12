package nadswap

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Extra struct {
	Reserve0           *uint256.Int `json:"r0"`
	Reserve1           *uint256.Int `json:"r1"`
	BlockTimestampLast uint32       `json:"ts"`
}

type StaticExtra struct {
	IsMemePair         bool           `json:"meme"`
	QuoteToken         common.Address `json:"qt"`
	CreatorFeeRate     uint16         `json:"cfr"`
	DexProtocolFeeRate uint16         `json:"dpfr"`
}

type SwapInfo struct {
	NewReserve0 *uint256.Int `json:"-"`
	NewReserve1 *uint256.Int `json:"-"`
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

type ReserveData struct {
	Reserve0           *uint256.Int
	Reserve1           *uint256.Int
	BlockTimestampLast uint32
}

func (r ReserveData) IsZero() bool {
	return r.Reserve0 == nil || r.Reserve1 == nil || (r.Reserve0.IsZero() && r.Reserve1.IsZero())
}
