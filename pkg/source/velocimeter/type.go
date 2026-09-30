package velocimeter

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type VelocimeterMetadata struct {
	Dec0 *big.Int
	Dec1 *big.Int
	R0   *big.Int
	R1   *big.Int
	St   bool
	T0   common.Address
	T1   common.Address
}

type StaticExtra struct {
	Stable bool `json:"stable"`
}

type MetaInfo struct {
	Stable      bool   `json:"stable"`
	BlockNumber uint64 `json:"blockNumber"`
}

type Reserves struct {
	Reserve0           *uint256.Int
	Reserve1           *uint256.Int
	BlockTimestampLast *uint256.Int
}

type Gas struct {
	Swap int64
}
