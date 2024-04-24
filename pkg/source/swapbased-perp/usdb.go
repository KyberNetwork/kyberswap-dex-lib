//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple USDB
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package swapbasedperp

import (
	"math/big"
)

type USDB struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdbMethodTotalSupply = "totalSupply"
)
