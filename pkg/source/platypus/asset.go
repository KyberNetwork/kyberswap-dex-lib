package platypus

import (
	"math/big"
)

func (a *Asset) addCash(amount *big.Int) {
	a.Cash = new(big.Int).Add(a.Cash, amount)
}

func (a *Asset) removeCash(amount *big.Int) {
	a.Cash = new(big.Int).Sub(a.Cash, amount)
}

func (a *Asset) addLiability(amount *big.Int) {
	a.Liability = new(big.Int).Add(a.Liability, amount)
}
