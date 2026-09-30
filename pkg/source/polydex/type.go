package polydex

import "github.com/holiman/uint256"

type Reserves struct {
	Reserve0           *uint256.Int
	Reserve1           *uint256.Int
	BlockTimestampLast uint32
}

type Metadata struct {
	Offset int `json:"offset"`
}
