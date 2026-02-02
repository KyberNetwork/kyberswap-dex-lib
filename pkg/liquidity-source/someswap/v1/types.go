package someswapv1

import (
	"math/big"

	"github.com/holiman/uint256"
)

type ReserveData struct {
	Reserve0           *big.Int `abi:"_r0"`
	Reserve1           *big.Int `abi:"_r1"`
	BlockTimestampLast uint32   `abi:"ts"`
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type Metadata struct {
	Offset int `json:"offset"`
}

type StaticExtra struct {
	WTokens [2]*uint256.Int `json:"ws"`
}
