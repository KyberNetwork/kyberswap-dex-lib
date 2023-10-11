package equalizer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type EqualizerMetadata struct {
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

type Reserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast *big.Int
}

type Gas struct {
	Swap int64
}
