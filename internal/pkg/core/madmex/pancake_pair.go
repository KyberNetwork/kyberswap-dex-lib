package madmex

import "math/big"

type PancakePair struct {
	Reserves      []*big.Int `json:"reserves"`
	TimestampLast uint32     `json:"timestampLast"`
}

func (p *PancakePair) GetReserves() (*big.Int, *big.Int, uint32) {
	return p.Reserves[0], p.Reserves[1], p.TimestampLast
}
