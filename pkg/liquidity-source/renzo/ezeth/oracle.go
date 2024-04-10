package ezeth

import "math/big"

type Oracle struct {
	Answer    *big.Int `json:"answer"`
	UpdatedAt *big.Int `json:"updatedAt"`
}

func (o *Oracle) LatestRoundData() (*big.Int, *big.Int) {
	return o.Answer, o.UpdatedAt
}
