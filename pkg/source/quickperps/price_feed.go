//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PriceFeed
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package quickperps

import (
	"math/big"
)

type PriceFeed struct {
	Price     *big.Int `json:"price"`
	Timestamp uint32   `json:"timestamp"`
}

const (
	priceFeedMethodRead = "read"
)
