package makerpsm

import (
	"errors"
	"math/big"
)

var (
	ErrDebtCeilingExceeded = errors.New("vat: debt ceiling exceeded")
)

// Vat implements Vat contract
// https://github.com/makerdao/dss/blob/master/src/vat.sol

// validateSellingGem implements validation when dart > 0 in frob
// https://github.com/makerdao/dss/blob/master/src/vat.sol#L143
func (v *Vat) validateSellingGem(dart *big.Int) error {
	newIlkArt := new(big.Int).Add(v.ILK.Art, dart)

	// newIlkArt * ilkRate > ilkLine
	if new(big.Int).Mul(newIlkArt, v.ILK.Rate).Cmp(v.ILK.Line) > 0 {
		return ErrDebtCeilingExceeded
	}

	newDebt := new(big.Int).Add(v.Debt, new(big.Int).Mul(v.ILK.Rate, dart))

	if newDebt.Cmp(v.Line) > 0 {
		return ErrDebtCeilingExceeded
	}

	return nil
}

// validateBuyingGem implements validation when dart < 0 in frob
// https://github.com/makerdao/dss/blob/master/src/vat.sol#L143
func (v *Vat) validateBuyingGem(dart *big.Int) error {
	if dart.Cmp(v.ILK.Art) > 0 {
		return ErrDebtCeilingExceeded
	}

	return nil
}

func (v *Vat) updateBalanceSellingGem(dart *big.Int) {
	v.ILK.Art = new(big.Int).Add(v.ILK.Art, dart)
	v.Debt = new(big.Int).Add(v.Debt, new(big.Int).Mul(v.ILK.Rate, dart))
}

func (v *Vat) updateBalanceBuyingGem(dart *big.Int) {
	v.ILK.Art = new(big.Int).Sub(v.ILK.Art, dart)
}
