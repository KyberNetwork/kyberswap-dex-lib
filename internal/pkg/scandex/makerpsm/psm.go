package makerpsm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PSM struct {
	TIn  *big.Int `json:"tIn"`
	TOut *big.Int `json:"tOut"`

	Vat        *Vat           `json:"vat"`
	VatAddress common.Address `json:"-"`

	ILK [32]byte `json:"-"`
}

const (
	PSMMethodTIn  = "tin"
	PSMMethodTOut = "tout"
	PSMMethodVat  = "vat"
	PSMMethodIlk  = "ilk"
)
