package makerpsm

import (
	"math/big"
)

type Vat struct {
	ILK ILK `json:"ilk"`

	Debt *big.Int `json:"debt"` // Total Dai Issued    [rad]
	Line *big.Int `json:"line"` // Total Debt Ceiling  [rad]
}

type ILK struct {
	Art  *big.Int `json:"art"`  // Total Normalised Debt     [wad]
	Rate *big.Int `json:"rate"` // Accumulated Rates         [ray]
	Line *big.Int `json:"line"` // Debt Ceiling              [rad]
	Spot *big.Int `json:"-"`
	Dust *big.Int `json:"-"`
}

const (
	VatMethodIlks = "ilks"
	VatMethodDebt = "debt"
	VatMethodLine = "Line"
)
