//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple USDQ
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package quickperps

import (
	"math/big"
)

type USDQ struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdqMethodTotalSupply = "totalSupply"
)
