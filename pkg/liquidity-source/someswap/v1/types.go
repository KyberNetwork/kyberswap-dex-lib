package someswapv1

import (
	"github.com/holiman/uint256"
)

type ReserveData struct {
	Reserve0 *uint256.Int `abi:"_r0"`
	Reserve1 *uint256.Int `abi:"_r1"`
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
