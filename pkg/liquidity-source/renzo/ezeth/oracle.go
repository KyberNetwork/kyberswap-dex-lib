package ezeth

import "math/big"

type Oracle struct {
	RoundId         *big.Int `json:"-"`
	Answer          *big.Int `json:"answer"`
	StartedAt       *big.Int `json:"-"`
	UpdatedAt       *big.Int `json:"updatedAt"`
	AnsweredInRound *big.Int `json:"-"`
}

func (o *Oracle) LatestRoundData() (*big.Int, *big.Int) {
	return o.Answer, o.UpdatedAt
}
