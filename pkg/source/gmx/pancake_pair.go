package gmx

import "math/big"

type PancakePair struct {
	Reserves      []*big.Int `json:"reserves"`
	TimestampLast uint32     `json:"timestampLast"`
}

func NewPancakePair() *PancakePair {
	return &PancakePair{}
}

const (
	pancakePairMethodGetReserves = "getReserves"
)
