package pool

import "math/big"

type TokenAmount struct {
	Token     string   `json:"token"`
	Amount    *big.Int `json:"amount"`
	AmountUsd float64  `json:"amountUsd"`
}

func (t *TokenAmount) CompareTo(other *TokenAmount) int {
	if other == nil || t.Token != other.Token {
		return -1
	}
	return t.Amount.Cmp(other.Amount)
}
