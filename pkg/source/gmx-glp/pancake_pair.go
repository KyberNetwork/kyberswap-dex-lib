package gmxglp

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

func (p *PancakePair) GetReserves() (*big.Int, *big.Int, uint32) {
	return p.Reserves[0], p.Reserves[1], p.TimestampLast
}
