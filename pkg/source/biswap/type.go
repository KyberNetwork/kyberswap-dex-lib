package biswap

import (
	"math/big"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type Reserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}
