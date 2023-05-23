package makerpsm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type DexConfig struct {
	Dai  Token       `json:"dai"`
	PSMs []PSMConfig `json:"psms"`
}

type PSMConfig struct {
	Address string `json:"address"`
	Gem     Token  `json:"gem"`
}

type Token struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
}

type PSM struct {
	TIn        *big.Int       `json:"tIn"`
	TOut       *big.Int       `json:"tOut"`
	Vat        *Vat           `json:"vat"`
	VatAddress common.Address `json:"-"`
	ILK        [32]byte       `json:"-"`
}

type Vat struct {
	ILK  ILK      `json:"ilk"`
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
