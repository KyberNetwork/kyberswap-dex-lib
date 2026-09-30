package biswap

import "github.com/holiman/uint256"

type Metadata struct {
	Offset int `json:"offset"`
}

type Reserves struct {
	Reserve0           *uint256.Int
	Reserve1           *uint256.Int
	BlockTimestampLast uint32
}
