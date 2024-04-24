//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple USDM
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package metavault

import (
	"math/big"
)

type USDM struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	USDMMethodTotalSupply = "totalSupply"
)
