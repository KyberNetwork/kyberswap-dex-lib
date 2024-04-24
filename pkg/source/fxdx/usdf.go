//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple USDF
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package fxdx

import "math/big"

type USDF struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdfMethodTotalSupply = "totalSupply"
)
