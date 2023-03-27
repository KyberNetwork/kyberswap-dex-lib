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
type Vat struct {
	Ilk ILK `json:"ilk"`

	Debt *big.Int `json:"debt"` // Total Dai Issued    [rad]
	Line *big.Int `json:"line"` // Total Debt Ceiling  [rad]
}

type ILK struct {
	Art  *big.Int `json:"art"`  // Total Normalised Debt     [wad]
	Rate *big.Int `json:"rate"` // Accumulated Rates         [ray]
	Line *big.Int `json:"line"` // Debt Ceiling              [rad]
}

// validateSellingGem implements validation when dart > 0 in frob
// https://github.com/makerdao/dss/blob/master/src/vat.sol#L143
func (v *Vat) validateSellingGem(dart *big.Int) error {
	newIlkArt := new(big.Int).Add(v.Ilk.Art, dart)

	// newIlkArt * ilkRate > ilkLine
	if new(big.Int).Mul(newIlkArt, v.Ilk.Rate).Cmp(v.Ilk.Line) > 0 {
		return ErrDebtCeilingExceeded
	}

	newDebt := new(big.Int).Add(v.Debt, new(big.Int).Mul(v.Ilk.Rate, dart))

	if newDebt.Cmp(v.Line) > 0 {
		return ErrDebtCeilingExceeded
	}

	return nil
}

// validateSellingGem implements validation when dart < 0 in frob
// https://github.com/makerdao/dss/blob/master/src/vat.sol#L143
func (v *Vat) validateBuyingGem(dart *big.Int) error {
	if dart.Cmp(v.Ilk.Art) > 0 {
		return ErrDebtCeilingExceeded
	}

	return nil
}

func (v *Vat) updateBalanceSellingGem(dart *big.Int) {
	v.Ilk.Art = new(big.Int).Add(v.Ilk.Art, dart)
	v.Debt = new(big.Int).Add(v.Debt, new(big.Int).Mul(v.Ilk.Rate, dart))
}

func (v *Vat) updateBalanceBuyingGem(dart *big.Int) {
	v.Ilk.Art = new(big.Int).Sub(v.Ilk.Art, dart)
}
