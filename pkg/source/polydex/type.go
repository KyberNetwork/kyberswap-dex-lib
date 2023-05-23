package polydex

import (
	"math/big"
)

type Reserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

type Metadata struct {
	Offset int `json:"offset"`
}
