//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Asset
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress

package vooi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Asset struct {
	Cash        *big.Int       `json:"cash"`
	Liability   *big.Int       `json:"liability"`
	MaxSupply   *big.Int       `json:"maxSupply"`
	TotalSupply *big.Int       `json:"totalSupply"`
	Decimals    uint8          `json:"decimals"`
	Token       common.Address `json:"token"`
	Active      bool           `json:"active"`
}
