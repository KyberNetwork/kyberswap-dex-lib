//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple USDG
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package zkerafinance

import (
	"math/big"
)

type USDG struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdgMethodTotalSupply = "totalSupply"
)
