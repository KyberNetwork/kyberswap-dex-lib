//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PSM Vat ILK Gas
//msgp:ignore DexConfig PSMConfig Token Extra
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress

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

	To18ConversionFactor *big.Int
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

type Extra struct {
	PSM PSM `json:"psm"`
}

type Gas struct {
	BuyGem  int64
	SellGem int64
}
