package platypus

import (
	"math/big"
)

type Asset struct {
	Id               string   `json:"id"`
	Decimals         uint8    `json:"decimals"`
	Cash             *big.Int `json:"cash"`
	Liability        *big.Int `json:"liability"`
	AggregateAccount string   `json:"aggregateAccount"`
	UnderlyingToken  string   `json:"underlyingToken"`
}

func (a *Asset) addCash(amount *big.Int) {
	a.Cash = new(big.Int).Add(a.Cash, amount)
}

func (a *Asset) removeCash(amount *big.Int) {
	a.Cash = new(big.Int).Sub(a.Cash, amount)
}

func (a *Asset) addLiability(amount *big.Int) {
	a.Liability = new(big.Int).Add(a.Liability, amount)
}
